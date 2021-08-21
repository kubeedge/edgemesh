package protocol

import (
	"fmt"
	"net"

	"k8s.io/klog/v2"
)

type ProtoName string

const (
	TCP  ProtoName = "tcp"
	UDP  ProtoName = "udp"
	SCTP ProtoName = "sctp"
)

type ProtoProxy interface {
	GetName() ProtoName
	GetProxyAddr() string
	SetListener(net.IP, int) error
}

type TCPProxy struct {
	Name     ProtoName
	Listener *net.TCPListener
}

func (tcp *TCPProxy) GetName() ProtoName {
	return TCP
}

func (tcp *TCPProxy) GetProxyAddr() string {
	return tcp.Listener.Addr().String()
}

func (tcp *TCPProxy) SetListener(ip net.IP, port int) error {
	tmpPort := 0
	listenAddr := &net.TCPAddr{
		IP:   ip,
		Port: port + tmpPort,
	}
	for {
		ln, err := net.ListenTCP("tcp", listenAddr)
		if err == nil {
			tcp.Listener = ln
			break
		}
		klog.Warningf("tcp proxy listen on address %s err: %v", listenAddr.String(), err)
		tmpPort++
		listenAddr = &net.TCPAddr{
			IP:   ip,
			Port: port + tmpPort,
		}
		if listenAddr.Port >= 65535 {
			return fmt.Errorf("max port limit 1-65535")
		}
	}

	return nil
}
