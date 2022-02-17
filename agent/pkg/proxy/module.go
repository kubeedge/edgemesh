package proxy

import (
	"fmt"

	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/edgemesh/agent/pkg/proxy/config"
	"github.com/kubeedge/edgemesh/common/informers"
	"github.com/kubeedge/edgemesh/common/modules"
	"github.com/kubeedge/edgemesh/common/util"
)

// EdgeProxy is used for traffic proxy
type EdgeProxy struct {
	Config      *config.EdgeProxyConfig
	ProxyServer *ProxyServer
	Socks5Proxy *Socks5Proxy
}

func newEdgeProxy(c *config.EdgeProxyConfig, ifm *informers.Manager) (proxy *EdgeProxy, err error) {
	proxy = &EdgeProxy{Config: c}
	if !c.Enable {
		return proxy, nil
	}

	// get proxy listen ip
	listenIP, err := util.GetInterfaceIP(proxy.Config.ListenInterface)
	if err != nil {
		return proxy, fmt.Errorf("get proxy listen ip err: %v", err)
	}

	// new proxy server
	proxy.ProxyServer, err = newProxyServer(NewDefaultKubeProxyConfiguration(listenIP.String()), false, ifm.GetKubeClient(), ifm.GetIstioClient())
	if err != nil {
		return proxy, fmt.Errorf("new proxy server err: %v", err)
	}

	// new socks5 proxy
	if proxy.Config.Socks5Proxy.Enable {
		proxy.Socks5Proxy, err = NewSocks5Proxy(listenIP, proxy.Config.Socks5Proxy.ListenPort, proxy.Config.NodeName, ifm.GetKubeClient())
		if err != nil {
			return proxy, fmt.Errorf("new socks5Proxy err: %w", err)
		}
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
