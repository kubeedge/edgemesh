package config

// EdgeProxyConfig indicates the edgeproxy config
type EdgeProxyConfig struct {
	// Enable indicates whether enable edgeproxy
	// default true
	Enable bool `json:"enable,omitempty"`
	// SubNet indicates the subnet of proxier, which equals to k8s service-cluster-ip-range
	// no need for user manual configuration
	SubNet string
	// ListenInterface indicates the listen interface of edgeproxy
	// do not allow users to configure manually
	ListenInterface string
	// ListenPort indicates the listen port of edgeproxy
	// default 40001
	ListenPort int `json:"listenPort,omitempty"`
}

func NewEdgeProxyConfig() *EdgeProxyConfig {
	return &EdgeProxyConfig{
		Enable:     true,
		ListenPort: 40001,
	}
}
