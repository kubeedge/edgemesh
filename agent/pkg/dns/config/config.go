package config

import "github.com/kubeedge/kubeedge/pkg/apis/componentconfig/cloudcore/v1alpha1"

// EdgeDNSConfig indicates the edgedns config
type EdgeDNSConfig struct {
	// Enable indicates whether enable edgedns
	// default false
	Enable bool `json:"enable,omitempty"`
	// ListenInterface indicates the listen interface of edgedns
	// do not allow users to configure manually
	ListenInterface string `json:"listenInterface,omitempty"`
	// ListenPort indicates the listen port of edgedns
	// default 53
	ListenPort int `json:"listenPort,omitempty"`
	// Mode is equivalent to CommonConfig.Mode
	// do not allow users to configure manually
	Mode string
	// KubeAPIConfig is equivalent to EdgeMeshAgentConfig.KubeAPIConfig
	// do not allow users to configure manually
	KubeAPIConfig *v1alpha1.KubeAPIConfig
	// CacheDNS indicates the nodelocal cache dns
	CacheDNS *CacheDNS `json:"cacheDNS,omitempty"`
}

type CacheDNS struct {
	// Enable indicates whether enable nodelocal cache dns
	// default false
	Enable bool `json:"enable,omitempty"`
	// AutoDetect indicates whether to automatically detect the
	// address of the upstream clusterDNS
	// default true
	AutoDetect bool `json:"autoDetect,omitempty"`
	// UpstreamServers indicates the upstream ClusterDNS addresses
	UpstreamServers []string `json:"upstreamServers,omitempty"`
	// CacheTTL indicates the time to live of a dns cache entry
	// default 30(second)
	CacheTTL int `json:"cacheTTL,omitempty"`
}

func NewEdgeDNSConfig() *EdgeDNSConfig {
	return &EdgeDNSConfig{
		Enable:     false,
		ListenPort: 53,
		CacheDNS: &CacheDNS{
			Enable:     false,
			AutoDetect: true,
			CacheTTL:   30,
		},
	}
}
