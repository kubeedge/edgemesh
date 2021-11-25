package config

// EdgeGatewayConfig indicates the edge gateway config
type EdgeGatewayConfig struct {
	// Enable indicates whether enable edge gateway
	// default false
	Enable bool `json:"enable,omitempty"`
	// NIC indicates the network interface controller that the edge gateway needs to listen to.
	// empty or "*" stands for all netcards. You can also specify network adapters such as "lo,eth0"
	// default "*"
	NIC string `json:"nic,omitempty"`
	// IncludeIP indicates the host IP that the edge gateway needs to listen to.
	// empty or "*" stands for all ips. You can also specify ips such as "192.168.1.56,10.3.2.1"
	// default "*"
	IncludeIP string `json:"includeIP,omitempty"`
	// ExcludeIP indicates the IP address that the edge gateway does not want to listen to.
	// empty or "*" stands for not exclude any ip. You can also specify ips such as "192.168.1.56,10.3.2.1"
	// default "*"
	ExcludeIP string `json:"excludeIP,omitempty"`
}

func NewEdgeGatewayConfig() *EdgeGatewayConfig {
	return &EdgeGatewayConfig{
		Enable:    false,
		NIC:       "*",
		IncludeIP: "*",
		ExcludeIP: "*",
	}
}
