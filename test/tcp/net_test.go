package functions

import (
	"k8s.io/klog/v2"
	"net"
	"testing"
)

func TestNet(t *testing.T) {

	host := "192.168.100.1"
	port := 31883

	addr := &net.TCPAddr{
		IP:   net.ParseIP(host),
		Port: port,
	}
	klog.Infof("l4 proxy get server address: %v", addr)
}
