package proxy

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"syscall"
	"unsafe"

	"k8s.io/klog/v2"

	"github.com/kubeedge/edgemesh/agent/pkg/chassis/protocol"
	"github.com/kubeedge/edgemesh/agent/pkg/chassis/protocol/http"
	"github.com/kubeedge/edgemesh/agent/pkg/chassis/protocol/tcp"
	"github.com/kubeedge/edgemesh/agent/pkg/proxy/controller"
)

const SoOriginalDst = 80

type sockAddr struct {
	family uint16
	data   [14]byte
}

func (proxy *EdgeProxy) Run() {
	// ensure ipatbles
	proxy.Proxier.Start()

	// start tcp proxy
	for {
		conn, err := proxy.TCPProxy.Listener.Accept()
		if err != nil {
			klog.Warningf("get tcp conn error: %v", err)
			continue
		}
		ip, port, err := realServerAddress(&conn)
		klog.Info("clusterIP: ", ip, ", servicePort: ", port)
		if err != nil {
			klog.Warningf("get real destination of tcp conn error: %v", err)
			conn.Close()
			continue
		}
		proto, err := proxy.newProtocolFromSock(ip, port, conn)
		if err != nil {
			klog.Warningf("get protocol from sock error: %v", err)
			conn.Close()
			continue
		}
		go proto.Process()
	}
}

// realServerAddress returns an intercepted connection's original destination.
func realServerAddress(conn *net.Conn) (string, int, error) {
	tcpConn, ok := (*conn).(*net.TCPConn)
	if !ok {
		return "", -1, fmt.Errorf("not a TCPConn")
	}

	file, err := tcpConn.File()
	if err != nil {
		return "", -1, err
	}
	defer file.Close()

	// To avoid potential problems from making the socket non-blocking.
	tcpConn.Close()
	*conn, err = net.FileConn(file)
	if err != nil {
		return "", -1, err
	}

	fd := file.Fd()

	var addr sockAddr
	size := uint32(unsafe.Sizeof(addr))
	err = getSockOpt(int(fd), syscall.SOL_IP, SoOriginalDst, uintptr(unsafe.Pointer(&addr)), &size)
	if err != nil {
		return "", -1, err
	}

	var ip net.IP
	switch addr.family {
	case syscall.AF_INET:
		ip = addr.data[2:6]
	default:
		return "", -1, fmt.Errorf("unrecognized address family")
	}

	port := int(addr.data[0])<<8 + int(addr.data[1])
	if err := syscall.SetNonblock(int(fd), true); err != nil {
		return "", -1, nil
	}

	return ip.String(), port, nil
}

func getSockOpt(s int, level int, name int, val uintptr, vallen *uint32) (err error) {
	_, _, e1 := syscall.Syscall6(syscall.SYS_GETSOCKOPT, uintptr(s), uintptr(level), uintptr(name), uintptr(val), uintptr(unsafe.Pointer(vallen)), 0)
	if e1 != 0 {
		err = e1
	}
	return
}

// newProtocolFromSock returns a protocol.Protocol interface if the ip is in proxier list
func (proxy *EdgeProxy) newProtocolFromSock(ip string, port int, conn net.Conn) (proto protocol.Protocol, err error) {
	svcPorts := controller.APIConn.GetSvcPorts(ip)
	protoName, svcName := getProtocol(svcPorts, port)
	if protoName == "" || svcName == "" {
		return nil, fmt.Errorf("protocol name: %s or svcName: %s is invalid", protoName, svcName)
	}

	svcNameSets := strings.Split(svcName, ".")
	if len(svcNameSets) != 2 {
		return nil, fmt.Errorf("invalid length %d after splitting svc name %s", len(svcNameSets), svcName)
	}
	namespace := svcNameSets[0]
	name := svcNameSets[1]

	switch protoName {
	case "http":
		proto = &http.HTTP{
			Conn:         conn,
			SvcName:      name,
			SvcNamespace: namespace,
			Port:         port,
		}
		err = nil
	case "tcp":
		proto = &tcp.TCP{
			Conn:         conn,
			SvcName:      name,
			SvcNamespace: namespace,
			Port:         port,
		}
		err = nil
	default:
		proto = nil
		err = fmt.Errorf("protocol: %s is not supported yet", protoName)
	}
	return
}

// getProtocol gets protocol name
func getProtocol(svcPorts string, port int) (string, string) {
	var protoName string
	sub := strings.Split(svcPorts, "|")
	n := len(sub)
	if n < 2 {
		return "", ""
	}
	svcName := sub[n-1]

	pstr := strconv.Itoa(port)
	if pstr == "" {
		return "", ""
	}
	for _, s := range sub {
		if strings.Contains(s, pstr) {
			protoName = strings.Split(s, ",")[0]
			break
		}
	}
	return protoName, svcName
}
