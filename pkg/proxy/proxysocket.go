package proxy

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/proxy"
	"k8s.io/kubernetes/pkg/proxy/userspace"

	"github.com/kubeedge/edgemesh/pkg/loadbalancer"
	netutil "github.com/kubeedge/edgemesh/pkg/util/net"
)

var (
	once                 sync.Once
	internalLoadBalancer *loadbalancer.LoadBalancer
)

func initLoadBalancer(loadbalancer *loadbalancer.LoadBalancer) {
	once.Do(func() {
		internalLoadBalancer = loadbalancer
	})
}

func newProxySocket(protocol v1.Protocol, ip net.IP, port int) (userspace.ProxySocket, error) {
	host := ""
	if ip != nil {
		host = ip.String()
	}

	switch strings.ToUpper(string(protocol)) {
	case "TCP":
		listener, err := net.Listen("tcp", net.JoinHostPort(host, strconv.Itoa(port)))
		if err != nil {
			return nil, err
		}
		return &tcpProxySocket{Listener: listener, port: port}, nil
	case "UDP":
		addr, err := net.ResolveUDPAddr("udp", net.JoinHostPort(host, strconv.Itoa(port)))
		if err != nil {
			return nil, err
		}
		conn, err := net.ListenUDP("udp", addr)
		if err != nil {
			return nil, err
		}
		return &udpProxySocket{UDPConn: conn, port: port}, nil
	case "SCTP":
		return nil, fmt.Errorf("SCTP is not supported for user space proxy")
	}
	return nil, fmt.Errorf("unknown protocol %q", protocol)
}

// tcpProxySocket implements ProxySocket.  Close() is implemented by net.Listener.  When Close() is called,
// no new connections are allowed but existing connections are left untouched.
type tcpProxySocket struct {
	net.Listener
	port int
}

func (tcp *tcpProxySocket) ListenPort() int {
	return tcp.port
}

func (tcp *tcpProxySocket) ProxyLoop(service proxy.ServicePortName, myInfo *userspace.ServiceInfo, loadBalancer userspace.LoadBalancer) {
	for {
		if !myInfo.IsAlive() {
			// The service port was closed or replaced.
			return
		}
		// Block until a connection is made.
		inConn, err := tcp.Accept()
		if err != nil {
			if netutil.IsTooManyFDsError(err) {
				panic("Accept failed: " + err.Error())
			}

			if netutil.IsClosedError(err) {
				return
			}
			if !myInfo.IsAlive() {
				// Then the service port was just closed so the accept failure is to be expected.
				return
			}
			klog.ErrorS(err, "Accept failed")
			continue
		}
		klog.V(3).InfoS("Accepted TCP connection from remote", "remoteAddress", inConn.RemoteAddr(), "localAddress", inConn.LocalAddr())
		outConn, err := internalLoadBalancer.TryConnectEndpoints(service, inConn.RemoteAddr(), "tcp", inConn, nil)
		if err != nil {
			klog.ErrorS(err, "Failed to connect to balancer")
			err = inConn.Close()
			if err != nil {
				klog.ErrorS(err, "close error")
			}
			continue
		}
		// Spin up an async copy loop.
		go netutil.ProxyConn(inConn, outConn)
	}
}

// udpProxySocket implements ProxySocket.  Close() is implemented by net.UDPConn.  When Close() is called,
// no new connections are allowed and existing connections are broken.
// TODO: We could lame-duck this ourselves, if it becomes important.
type udpProxySocket struct {
	*net.UDPConn
	port int
}

func (udp *udpProxySocket) ListenPort() int {
	return udp.port
}

func (udp *udpProxySocket) Addr() net.Addr {
	return udp.LocalAddr()
}

func (udp *udpProxySocket) ProxyLoop(service proxy.ServicePortName, myInfo *userspace.ServiceInfo, loadBalancer userspace.LoadBalancer) {
	var buffer [4096]byte // 4KiB should be enough for most whole-packets
	for {
		if !myInfo.IsAlive() {
			// The service port was closed or replaced.
			break
		}

		// Block until data arrives.
		// TODO: Accumulate a histogram of n or something, to fine tune the buffer size.
		n, cliAddr, err := udp.ReadFrom(buffer[0:])
		if err != nil {
			if e, ok := err.(net.Error); ok {
				if e.Temporary() {
					klog.V(1).ErrorS(err, "ReadFrom had a temporary failure")
					continue
				}
			}
			if !netutil.IsClosedError(err) && !netutil.IsStreamResetError(err) {
				klog.ErrorS(err, "ReadFrom failed, exiting ProxyLoop")
			}
			break
		}
		// If this is a client we know already, reuse the connection and goroutine.
		svrConn, err := udp.getBackendConn(myInfo.ActiveClients, cliAddr, service, myInfo.Timeout)
		if err != nil {
			continue
		}
		// TODO: It would be nice to let the goroutine handle this write, but we don't
		// really want to copy the buffer.  We could do a pool of buffers or something.
		_, err = svrConn.Write(buffer[0:n])
		if err != nil {
			if !netutil.IsTimeoutError(err) {
				klog.ErrorS(err, "Write failed")
				// TODO: Maybe tear down the goroutine for this client/server pair?
			}
			continue
		}
		err = svrConn.SetDeadline(time.Now().Add(myInfo.Timeout))
		if err != nil {
			klog.ErrorS(err, "SetDeadline failed")
			continue
		}
	}
}

func (udp *udpProxySocket) getBackendConn(activeClients *userspace.ClientCache, cliAddr net.Addr, service proxy.ServicePortName, timeout time.Duration) (net.Conn, error) {
	activeClients.Mu.Lock()
	defer activeClients.Mu.Unlock()

	svrConn, found := activeClients.Clients[cliAddr.String()]
	if !found {
		// TODO: This could spin up a new goroutine to make the outbound connection,
		// and keep accepting inbound traffic.
		klog.V(3).InfoS("New UDP connection from client", "address", cliAddr)
		var err error
		svrConn, err = internalLoadBalancer.TryConnectEndpoints(service, cliAddr, "udp", nil, nil)
		if err != nil {
			return nil, err
		}
		if err = svrConn.SetDeadline(time.Now().Add(timeout)); err != nil {
			klog.ErrorS(err, "SetDeadline failed")
			return nil, err
		}
		activeClients.Clients[cliAddr.String()] = svrConn
		go func(cliAddr net.Addr, svrConn net.Conn, activeClients *userspace.ClientCache, timeout time.Duration) {
			defer runtime.HandleCrash()
			udp.proxyClient(cliAddr, svrConn, activeClients, timeout)
		}(cliAddr, svrConn, activeClients, timeout)
	}
	return svrConn, nil
}

// This function is expected to be called as a goroutine.
// TODO: Track and log bytes copied, like TCP
func (udp *udpProxySocket) proxyClient(cliAddr net.Addr, svrConn net.Conn, activeClients *userspace.ClientCache, timeout time.Duration) {
	defer svrConn.Close()
	var buffer [4096]byte
	for {
		n, err := svrConn.Read(buffer[0:])
		if err != nil {
			if !netutil.IsTimeoutError(err) && !netutil.IsEOFError(err) {
				klog.ErrorS(err, "Read failed")
			}
			break
		}
		err = svrConn.SetDeadline(time.Now().Add(timeout))
		if err != nil {
			klog.ErrorS(err, "SetDeadline failed")
			break
		}
		_, err = udp.WriteTo(buffer[0:n], cliAddr)
		if err != nil {
			if !netutil.IsTimeoutError(err) {
				klog.ErrorS(err, "WriteTo failed")
			}
			break
		}
	}
	activeClients.Mu.Lock()
	delete(activeClients.Clients, cliAddr.String())
	activeClients.Mu.Unlock()
}
