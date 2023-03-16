package gateway

import (
	"net"
	"net/http"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"

	netutil "github.com/kubeedge/edgemesh/pkg/util/net"
)

type TCP struct {
	Conn         net.Conn
	SvcName      string
	SvcNamespace string
	SvcPort      int
	UpgradeReq   *http.Request
}

// Process process
func (p *TCP) Process() {
	// find a service
	svcName := types.NamespacedName{Namespace: p.SvcNamespace, Name: p.SvcName}
	svcPort, ok := internalLoadBalancer.GetServicePortName(svcName, p.SvcPort)
	if !ok {
		klog.Errorf("destination service %s not found in cache", svcName)
		return
	}
	klog.Infof("destination service is %s", svcPort)

	klog.V(3).InfoS("Accepted TCP connection from remote", "remoteAddress", p.Conn.RemoteAddr(), "localAddress", p.Conn.LocalAddr())
	outConn, err := internalLoadBalancer.TryConnectEndpoints(svcPort, p.Conn.RemoteAddr(), "tcp", p.Conn, p.UpgradeReq)
	if err != nil {
		klog.ErrorS(err, "Failed to connect to balancer")
		err = p.Conn.Close()
		if err != nil {
			klog.Errorf("close conn err: %v", err)
		}
		return
	}

	go netutil.ProxyConn(p.Conn, outConn)
}
