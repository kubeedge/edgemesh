package config

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
		AutoDetect: true,
		CacheTTL:   30,
	}
}
