package proxy

import (
	"context"
	"fmt"
	"github.com/kubeedge/edgemesh/agent/pkg/proxy/protocol"
	"github.com/kubeedge/edgemesh/agent/pkg/tunnel"
	"github.com/kubeedge/edgemesh/common/constants"
	"io"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	"net"
	"strings"
	"sync"
)

type Socks5Proxy struct {
	TCPProxy   *protocol.TCPProxy
	kubeClient kubernetes.Interface
	NodeName   string
}

const (
	labelKubeedge string = "kubeedge=edgemesh-agent"
	agentPodName  string = "edgemesh-agent"
)

func (s *Socks5Proxy) Start() {
	go func() {
		for {
			conn, err := s.TCPProxy.Listener.Accept()
			if err != nil {
				klog.Warningf("get socks5 tcp conn error: %v", err)
				continue
			}
			go s.HandleSocksProxy(conn)
		}
	}()
}

func NewSocks5Proxy(ip net.IP, port int, NodeName string, kubeClient kubernetes.Interface) (socks5Proxy *Socks5Proxy, err error) {
	socks := &Socks5Proxy{
		kubeClient: kubeClient,
		TCPProxy:   &protocol.TCPProxy{Name: protocol.TCP},
		NodeName:   NodeName,
	}

	if err := socks.TCPProxy.SetListener(ip, port); err != nil {
		return socks, fmt.Errorf("set socks5 proxy err: %v, host: %s, port: %d", err, ip, port)
	}

	return socks, nil
}

func pipe(dst io.WriteCloser, src io.ReadCloser, closeOnce *sync.Once) {
	_, err := io.Copy(dst, src)
	if err != nil && err != io.EOF && !strings.Contains(err.Error(), constants.ConnectionClosed) && !strings.Contains(err.Error(), constants.StreamReset) {
		klog.Errorf("io copy between proxy and client error: %v", err)
	}
	closeOnce.Do(func() {
		dst.Close()
		src.Close()
	})
}

func (s *Socks5Proxy) HandleSocksProxy(conn net.Conn) {
	if conn == nil {
		return
	}
	defer conn.Close()

	var b [1024]byte
	n, err := conn.Read(b[:])
	if err != nil {
		klog.Errorf("Unable to get data from client connection, err: %v", err)
		return
	}

	if b[0] == 0x05 {
		conn.Write([]byte{0x05, 0x00})
		n, err = conn.Read(b[:])
		var host string
		var port int32
		switch b[3] {
		case 0x01: //IP V4
			host = net.IPv4(b[4], b[5], b[6], b[7]).String()
		case 0x03: //domain
			host = string(b[5 : n-2])
		case 0x04: //IP V6
			host = net.IP{b[4], b[5], b[6], b[7], b[8], b[9], b[10], b[11], b[12], b[13], b[14], b[15], b[16], b[17], b[18], b[19]}.String()
		}
		port = int32(int(b[n-2])<<8 | int(b[n-1]))
		klog.Infof("Successfully get data from socks5, host:%s, port: %d", host, port)

		if b[3] != 0x03 || host == s.NodeName {
			klog.Warningf("Connecting to the local computer and connecting via IP are not supported. host: %s, port: %d, localNodeName: %s", host, port, s.NodeName)
			return
		}

		targetIP, err := s.getTargetIpByNodeName(host)
		if err != nil {
			klog.Errorf("Unable to get destination IP, %v", err)
			return
		}
		klog.Info("Successfully get destination IP. NodeIP: ", targetIP, ", Port: ", port)

		proxyToRemote(host, targetIP, port, conn)
	}
}

func proxyToRemote(host string, targetIP string, port int32, conn net.Conn)  {
	stream, err := tunnel.Agent.TCPProxySvc.GetProxyStream(host, targetIP, port)
	if err != nil {
		klog.Errorf("l4 proxy get proxy stream from %s error: %v", host, err)
		return
	}

	klog.Infof("l4 proxy start proxy data between tcpserver %v", host)
	conn.Write([]byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})

	closeOnce := &sync.Once{}
	go pipe(conn, stream, closeOnce)
	pipe(stream, conn, closeOnce)

	klog.Infof("Success proxy to %v", host)
}

func (s *Socks5Proxy) getTargetIpByNodeName(nodeName string) (targetIP string, err error) {
	pods, err := s.kubeClient.CoreV1().Pods(constants.EdgeMeshNamespace).List(context.Background(), metav1.ListOptions{FieldSelector: "spec.nodeName=" + nodeName, LabelSelector: labelKubeedge})
	if err != nil {
		return "", err
	}
	ip, err := "", fmt.Errorf("edgemesh agent not found on node [%s]", nodeName)
	for _, pod := range pods.Items {
		if strings.Contains(pod.Name, agentPodName) {
			ip = pod.Status.PodIP
			err = nil
		}
	}

	return ip, err
}
