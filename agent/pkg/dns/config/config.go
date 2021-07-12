package config

// EdgeDNSConfig indicates the edgedns config
type EdgeDNSConfig struct {
	// Enable indicates whether enable edgedns
	// default true
	Enable bool `json:"enable,omitempty"`
	// ListenInterface indicates the listen interface of edgedns
	// default "docker0"
	ListenInterface string `json:"listenInterface,omitempty"`
	// ListenPort indicates the listen port of edgedns
	// default 53
	ListenPort int `json:"listenPort,omitempty"`
}

func NewEdgeDNSConfig() *EdgeDNSConfig {
	return &EdgeDNSConfig{
		Enable:          true,
		ListenInterface: "docker0",
		ListenPort:      53,
	}
}
