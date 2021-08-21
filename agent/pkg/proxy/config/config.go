package config

// EdgeProxyConfig indicates the edgeproxy config
type EdgeProxyConfig struct {
	// Enable indicates whether enable edgeproxy
	// default true
	Enable bool `json:"enable,omitempty"`
	// SubNet indicates the subnet of proxier
	// default "10.96.0.0/12", equals to k8s default service-cluster-ip-range
	SubNet string `json:"subNet,omitempty"`
	// ListenInterface indicates the listen interface of edgeproxy
	// default "docker0"
	ListenInterface string `json:"listenInterface,omitempty"`
	// ListenPort indicates the listen port of edgeproxy
	// default 40001
	ListenPort int `json:"listenPort,omitempty"`
}

func NewEdgeProxyConfig() *EdgeProxyConfig {
	return &EdgeProxyConfig{
		Enable:          true,
		SubNet:          "10.96.0.0/12",
		ListenInterface: "docker0",
		ListenPort:      40001,
	}
}
