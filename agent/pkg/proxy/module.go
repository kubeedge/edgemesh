package proxy

import (
	"fmt"

	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/edgemesh/agent/pkg/proxy/config"
	"github.com/kubeedge/edgemesh/agent/pkg/proxy/controller"
	"github.com/kubeedge/edgemesh/agent/pkg/proxy/protocol"
	"github.com/kubeedge/edgemesh/common/informers"
	"github.com/kubeedge/edgemesh/common/modules"
	"github.com/kubeedge/edgemesh/common/util"
)

// EdgeProxy is used for traffic proxy
type EdgeProxy struct {
	Config   *config.EdgeProxyConfig
	TCPProxy *protocol.TCPProxy
	Proxier  *Proxier
	// TODO(Poorunga) realize the proxy of UDP protocol
	// UDPProxy *protocol.UDPProxy
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

	// get tcp proxy
	proxy.TCPProxy = &protocol.TCPProxy{Name: protocol.TCP}
	if err := proxy.TCPProxy.SetListener(listenIP, proxy.Config.ListenPort); err != nil {
		return proxy, fmt.Errorf("set tcp proxy err: %v", err)
	}

	// new proxier
	protoProxies := []protocol.ProtoProxy{proxy.TCPProxy}
	proxy.Proxier, err = NewProxier(proxy.Config.SubNet, protoProxies, ifm.GetKubeClient())
	if err != nil {
		return proxy, fmt.Errorf("new proxier err: %v", err)
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
