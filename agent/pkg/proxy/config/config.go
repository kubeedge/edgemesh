package config

import (
	"github.com/kubeedge/edgemesh/common/util"
)

// EdgeProxyConfig indicates the edgeproxy config
type EdgeProxyConfig struct {
	// Enable indicates whether enable edgeproxy
	// default false
	Enable bool `json:"enable,omitempty"`
	// SubNet indicates the subnet of proxier, which equals to k8s service-cluster-ip-range
	SubNet string `json:"subNet,omitempty"`
	// ListenInterface indicates the listen interface of edgeproxy
	// do not allow users to configure manually
	ListenInterface string `json:"listenInterface,omitempty"`
	// ListenPort indicates the listen port of edgeproxy
	// default 40001
	ListenPort int `json:"listenPort,omitempty"`
	// Socks5Proxy indicates the socks5 proxy config
	Socks5Proxy *Socks5Proxy `json:"socks5Proxy,omitempty"`
	// NodeName indicates name of host
	NodeName string `json:"nodeName,omitempty"`
}

// Socks5Proxy indicates the socks5 proxy config
type Socks5Proxy struct {
	// Enable indicates whether enable socks5 proxy server
	// default false
	Enable bool `json:"enable,omitempty"`
	// ListenPort indicates the listen port of Socks5Proxy
	// default 10800
	ListenPort int `json:"listenPort,omitempty"`
}

func NewEdgeProxyConfig() *EdgeProxyConfig {
	return &EdgeProxyConfig{
		Enable:     false,
		ListenPort: 40001,
		NodeName:   util.FetchNodeName(),
		Socks5Proxy: &Socks5Proxy{
			Enable:     false,
			ListenPort: 10800,
		},
	}
}
