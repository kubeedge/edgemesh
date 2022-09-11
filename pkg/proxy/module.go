package proxy

import (
	"fmt"

	"github.com/kubeedge/beehive/pkg/core"

	"github.com/kubeedge/edgemesh/pkg/apis/config/defaults"
	"github.com/kubeedge/edgemesh/pkg/apis/config/v1alpha1"
	"github.com/kubeedge/edgemesh/pkg/common/informers"
	"github.com/kubeedge/edgemesh/pkg/common/util"
)

// EdgeProxy is used for traffic proxy
type EdgeProxy struct {
	Config      *v1alpha1.EdgeProxyConfig
	ProxyServer *ProxyServer
	Socks5Proxy *Socks5Proxy
}

// Name of edgeproxy
func (proxy *EdgeProxy) Name() string {
	return defaults.EdgeProxyModuleName
}

// Group of edgeproxy
func (proxy *EdgeProxy) Group() string {
	return defaults.EdgeProxyModuleName
}

// Enable indicates whether enable this module
func (proxy *EdgeProxy) Enable() bool {
	return proxy.Config.Enable
}

// Start edgeproxy
func (proxy *EdgeProxy) Start() {
	proxy.Run()
}

// Register register edgeproxy to beehive modules
func Register(c *v1alpha1.EdgeProxyConfig, ifm *informers.Manager) error {
	proxy, err := newEdgeProxy(c, ifm)
	if err != nil {
		return fmt.Errorf("register module edgeproxy error: %v", err)
	}
	core.Register(proxy)
	return nil
}

func newEdgeProxy(c *v1alpha1.EdgeProxyConfig, ifm *informers.Manager) (*EdgeProxy, error) {
	if !c.Enable {
		return &EdgeProxy{Config: c}, nil
	}

	// get proxy listen ip
	listenIP, err := util.GetInterfaceIP(c.ListenInterface)
	if err != nil {
		return nil, fmt.Errorf("get proxy listen ip err: %v", err)
	}

	// new proxy server
	proxyServer, err := newProxyServer(NewDefaultKubeProxyConfiguration(listenIP.String()), false, ifm.GetKubeClient(), ifm.GetIstioClient())
	if err != nil {
		return nil, fmt.Errorf("new proxy server err: %v", err)
	}

	// new socks5 proxy
	var socks5Proxy *Socks5Proxy
	if c.Socks5Proxy.Enable {
		socks5Proxy, err = NewSocks5Proxy(listenIP, c.Socks5Proxy.ListenPort, ifm.GetKubeClient())
		if err != nil {
			return nil, fmt.Errorf("new socks5Proxy err: %w", err)
		}
	}

	return &EdgeProxy{
		Config:      c,
		ProxyServer: proxyServer,
		Socks5Proxy: socks5Proxy,
	}, nil
}
