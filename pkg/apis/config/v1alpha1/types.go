package v1alpha1

import (
	"github.com/kubeedge/edgemesh/pkg/apis/config/defaults"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/cloudcore/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EdgeMeshAgentConfig indicates the config of edgeMeshAgent which get from edgeMeshAgent config file
type EdgeMeshAgentConfig struct {
	metav1.TypeMeta
	// KubeAPIConfig indicates the kubernetes cluster info which edgeMeshAgent will connected
	// +Required
	KubeAPIConfig *v1alpha1.KubeAPIConfig `json:"kubeAPIConfig,omitempty"`
	// CommonConfig indicates common config for all modules
	// +Required
	CommonConfig *CommonConfig `json:"commonConfig,omitempty"`
	// Modules indicates edgeMeshAgent modules config
	// +Required
	Modules *Modules `json:"modules,omitempty"`
}

// CommonConfig defines some common configuration items
type CommonConfig struct {
	// Mode indicates the current running mode of edgemesh-agent
	// do not allow users to configure manually
	// default "CloudMode"
	Mode string `json:"mode,omitempty"`
	// DummyDeviceName indicates the name of the dummy device will be created
	// default edgemesh0
	DummyDeviceName string `json:"dummyDeviceName,omitempty"`
	// DummyDeviceIP indicates the IP bound to the dummy device
	// default "169.254.96.16"
	DummyDeviceIP string `json:"dummyDeviceIP,omitempty"`
}

// Modules indicates the modules of edgeMeshAgent will be use
type Modules struct {
	// EdgeDNSConfig indicates edgedns module config
	EdgeDNSConfig *EdgeDNSConfig `json:"edgeDNS,omitempty"`
	// EdgeProxyConfig indicates edgeproxy module config
	EdgeProxyConfig *EdgeProxyConfig `json:"edgeProxy,omitempty"`
	// EdgeGatewayConfig indicates edgegateway module config
	EdgeGatewayConfig *EdgeGatewayConfig `json:"edgeGateway,omitempty"`
	// EdgeTunnelConfig indicates tunnel module config
	EdgeTunnelConfig *EdgeTunnelConfig `json:"tunnel,omitempty"`
}

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

// EdgeProxyConfig indicates the edgeproxy config
type EdgeProxyConfig struct {
	// Enable indicates whether enable edgeproxy
	// default false
	Enable bool `json:"enable,omitempty"`
	// ListenInterface indicates the listen interface of edgeproxy
	// do not allow users to configure manually
	ListenInterface string `json:"listenInterface,omitempty"`
	// Socks5Proxy indicates the socks5 proxy config
	Socks5Proxy *Socks5Proxy `json:"socks5Proxy,omitempty"`
}

// Socks5Proxy indicates the socks5 proxy config
type Socks5Proxy struct {
	// Enable indicates whether enable socks5 proxy server
	// default false
	Enable bool `json:"enable,omitempty"`
	// ListenPort indicates the listen port of Socks5Proxy
	// default 10800
	ListenPort int `json:"listenPort,omitempty"`
	// NodeName indicates name of host
	NodeName string `json:"nodeName,omitempty"`
	// Namespace indicates namespace of host
	Namespace string `json:"namespace,omitempty"`
}

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
	// GoChassisConfig defines some configurations related to go-chassis
	// +Required
	GoChassisConfig *GoChassisConfig `json:"goChassisConfig,omitempty"`
}

// GoChassisConfig defines some configurations related to go-chassis
type GoChassisConfig struct {
	// Protocol indicates the network protocols config supported in edgemesh
	Protocol *Protocol `json:"protocol,omitempty"`
	// LoadBalancer indicates the load balance strategy
	LoadBalancer *LoadBalancer `json:"loadBalancer,omitempty"`
}

// Protocol indicates the network protocols config supported in edgemesh
type Protocol struct {
	// TCPBufferSize indicates 4-layer tcp buffer size
	// default 8192
	TCPBufferSize int `json:"tcpBufferSize,omitempty"`
	// TCPClientTimeout indicates 4-layer tcp client timeout, the unit is second.
	// default 2
	TCPClientTimeout int `json:"tcpClientTimeout,omitempty"`
	// TCPReconnectTimes indicates 4-layer tcp reconnect times
	// default 3
	TCPReconnectTimes int `json:"tcpReconnectTimes,omitempty"`
	// NodeName indicates the node name of edgemesh agent
	NodeName string `json:"nodeName,omitempty"`
}

// LoadBalancer indicates the loadbalance strategy in edgemesh
type LoadBalancer struct {
	// DefaultLBStrategy indicates default load balance strategy name
	// default "RoundRobin"
	DefaultLBStrategy string `json:"defaultLBStrategy,omitempty"`
	// SupportedLBStrategies indicates supported load balance strategies name
	// default []string{"RoundRobin", "Random", "ConsistentHash"}
	SupportedLBStrategies []string `json:"supportLBStrategies,omitempty"`
	// ConsistentHash indicates the extension of the go-chassis loadbalancer
	ConsistentHash *ConsistentHash `json:"consistentHash,omitempty"`
}

// ConsistentHash strategy is an extension of the go-chassis loadbalancer
// For more information about the consistentHash algorithm, please take a look at
// https://research.googleblog.com/2017/04/consistent-hashing-with-bounded-loads.html
type ConsistentHash struct {
	// PartitionCount indicates the hash ring partition count
	// default 100
	PartitionCount int `json:"partitionCount,omitempty"`
	// ReplicationFactor indicates the hash ring replication factor
	// default 10
	ReplicationFactor int `json:"replicationFactor,omitempty"`
	// Load indicates the hash ring bounded loads
	// default 1.25
	Load float64 `json:"load,omitempty"`
}

type EdgeTunnelConfig struct {
	// Enable indicates whether EdgeTunnel is enabled,
	// if set to false (for debugging etc.), skip checking other EdgeTunnel configs.
	// default false
	Enable bool `json:"enable,omitempty"`
	// Mode indicates EdgeTunnel running mode
	// default ServerAndClient
	Mode defaults.TunnelMode `json:"mode,omitempty"`
	// NodeName indicates the node name of EdgeTunnel
	NodeName string `json:"nodeName,omitempty"`
	// ListenPort indicates the listen port of EdgeTunnel
	// default 20006
	ListenPort int `json:"listenPort,omitempty"`
	// EnableHolePunch indicates whether p2p hole punching feature is enabled,
	// default true
	EnableHolePunch bool `json:"enableHolePunch,omitempty"`
	// Transport indicates the transport protocol used by the p2p tunnel
	// default tcp
	Transport string `json:"transport,omitempty"`
	// Rendezvous unique string to identify group of libp2p nodes
	// default EDGEMESH_PLAYGOUND
	Rendezvous string `json:"rendezvous,omitempty"`
	// RelayNodes indicates some nodes that can become libp2p relay nodes
	RelayNodes []*RelayNode `json:"relayNodes,omitempty"`
	// EnableIpfsLog open ipfs log info
	// default false
	EnableIpfsLog bool `json:"enableIpfsLog,omitempty"`
	// MaxCandidates sets the number of relay candidates that we buffer.
	// default 5
	MaxCandidates int `json:"maxCandidates,omitempty"`
	// HeartbeatPeriod indicates the heartbeat period to keep connected with the relay peers (unit second)
	// default 120
	HeartbeatPeriod int `json:"heartbeatPeriod,omitempty"`
	// FinderPeriod indicates the execution period of the relay finder (unit second)
	// default 60
	FinderPeriod int `json:"finderPeriod,omitempty"`
}

type RelayNode struct {
	// NodeName indicates the relay node name, which is the same as the node name of Kubernetes
	NodeName string `json:"nodeName,omitempty"`
	// AdvertiseAddress sets the IP address for the relay node to advertise
	AdvertiseAddress []string `json:"advertiseAddress,omitempty"`
}
