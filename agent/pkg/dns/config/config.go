package config

// EdgeDNSConfig indicates the edgedns config
type EdgeDNSConfig struct {
	// Enable indicates whether enable edgedns
	// default true
	Enable bool `json:"enable,omitempty"`
	// ListenPort indicates the listen port of edgedns
	// default 53
	ListenPort int `json:"listenPort,omitempty"`
}

func NewEdgeDNSConfig() *EdgeDNSConfig {
	return &EdgeDNSConfig{
		Enable:     true,
		ListenPort: 53,
	}
}
