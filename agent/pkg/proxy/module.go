package proxy

import (
	"fmt"
	"net"

	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/edgemesh/agent/pkg/proxy/config"
	"github.com/kubeedge/edgemesh/agent/pkg/proxy/controller"
	"github.com/kubeedge/edgemesh/common/informers"
	"github.com/kubeedge/edgemesh/common/modules"
	"github.com/kubeedge/edgemesh/common/util"
)

// EdgeProxy is used for traffic proxy
type EdgeProxy struct {
	Config   *config.EdgeProxyConfig
	Listener *net.TCPListener
	Proxier  *Proxier
}

func newEdgeProxy(c *config.EdgeProxyConfig, ifm *informers.Manager) (proxy *EdgeProxy, err error) {
	proxy = &EdgeProxy{Config: c}
	if !c.Enable {
		return proxy, nil
	}

	// init proxy controller
	controller.Init(ifm)

	// get proxy listen ip
	listenIP, err := util.GetInterfaceIP(proxy.Config.ListenInterface)
	if err != nil {
		return proxy, fmt.Errorf("get proxy listen ip err: %v", err)
	}

	// get tcp listener
	tmpPort := 0
	listenAddr := &net.TCPAddr{
		IP:   listenIP,
		Port: proxy.Config.ListenPort + tmpPort,
	}
	for {
		ln, err := net.ListenTCP("tcp", listenAddr)
		if err == nil {
			proxy.Listener = ln
			break
		}
		klog.Warningf("listen on address %v error: %v", listenAddr, err)
		tmpPort++
		listenAddr = &net.TCPAddr{
			IP:   listenIP,
			Port: proxy.Config.ListenPort + tmpPort,
		}
	}

	// new proxier
	proxy.Proxier, err = newProxier(proxy.Config.SubNet, proxy.Config.ListenInterface,
		listenIP, proxy.Config.ListenPort)
	if err != nil {
		return proxy, fmt.Errorf("new proxier error: %v", err)
	}

	return proxy, nil
}

// Register register edgeproxy to beehive modules
func Register(c *config.EdgeProxyConfig, ifm *informers.Manager) error {
	proxy, err := newEdgeProxy(c, ifm)
	if err != nil {
		return fmt.Errorf("register module edgeproxy error: %v", err)
	}
	core.Register(proxy)
	return nil
}

// Name of edgeproxy
func (proxy *EdgeProxy) Name() string {
	return modules.EdgeProxyModuleName
}

// Group of edgeproxy
func (proxy *EdgeProxy) Group() string {
	return modules.EdgeProxyModuleName
}

// Enable indicates whether enable this module
func (proxy *EdgeProxy) Enable() bool {
	return proxy.Config.Enable
}

// Start edgeproxy
func (proxy *EdgeProxy) Start() {
	proxy.Run()
}
