package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeedge/edgemesh/pkg/apis/config/defaults"
	cloudcorev1alpha1 "github.com/kubeedge/kubeedge/pkg/apis/componentconfig/cloudcore/v1alpha1"
)

// EdgeMeshAgentConfig indicates the config of EdgeMeshAgent which get from EdgeMeshAgent config file
type EdgeMeshAgentConfig struct {
	metav1.TypeMeta
	// KubeAPIConfig indicates the kubernetes cluster info which EdgeMeshAgent will connect
	// +Required
	KubeAPIConfig *KubeAPIConfig `json:"kubeAPIConfig,omitempty"`
	// CommonConfig indicates common config for all modules
	// +Required
	CommonConfig *CommonConfig `json:"commonConfig,omitempty"`
	// Modules indicates EdgeMeshAgent modules config
	// +Required
	Modules *AgentModules `json:"modules,omitempty"`
}

// AgentModules indicates the modules of EdgeMeshAgent will be use
type AgentModules struct {
	// EdgeDNSConfig indicates EdgeDNS module config
	EdgeDNSConfig *EdgeDNSConfig `json:"edgeDNS,omitempty"`
	// EdgeProxyConfig indicates EdgeProxy module config
	EdgeProxyConfig *EdgeProxyConfig `json:"edgeProxy,omitempty"`
	// EdgeTunnelConfig indicates EdgeTunnel module config
	EdgeTunnelConfig *EdgeTunnelConfig `json:"edgeTunnel,omitempty"`
}

// EdgeMeshGatewayConfig indicates the config of EdgeMeshGateway which get from EdgeMeshGateway config file
type EdgeMeshGatewayConfig struct {
	metav1.TypeMeta
	// KubeAPIConfig indicates the kubernetes cluster info which EdgeMeshGateway will connect
	// +Required
	KubeAPIConfig *KubeAPIConfig `json:"kubeAPIConfig,omitempty"`
	// Modules indicates EdgeMeshAgent modules config
	// +Required
	Modules *GatewayModules `json:"modules,omitempty"`
}

// GatewayModules indicates the modules of EdgeMeshGateway will be use
type GatewayModules struct {
	// EdgeGatewayConfig indicates EdgeGateway module config
	EdgeGatewayConfig *EdgeGatewayConfig `json:"edgeGateway,omitempty"`
	// EdgeTunnelConfig indicates EdgeTunnel module config
	EdgeTunnelConfig *EdgeTunnelConfig `json:"edgeTunnel,omitempty"`
}

// KubeAPIConfig indicates the configuration for interacting with k8s server
type KubeAPIConfig struct {
	cloudcorev1alpha1.KubeAPIConfig
	// Mode indicates the current running mode of container
	// do not allow users to configure manually
	// options DebugMode, CloudMode and EdgeMode
	Mode defaults.RunningMode
	// MetaServerAddress indicates the EdgeCore's metaServer address
	// default http://127.0.0.1:10550
	MetaServerAddress string `json:"metaServerAddress,omitempty"`
}

// CommonConfig defines some common configuration items
type CommonConfig struct {
	// BridgeDeviceName indicates the name of the bridge device will be created
	// default edgemesh0
	BridgeDeviceName string `json:"bridgeDeviceName,omitempty"`
	// BridgeDeviceIP indicates the IP bound to the bridge device
	// default "169.254.96.16"
	BridgeDeviceIP string `json:"bridgeDeviceIP,omitempty"`
}

// EdgeDNSConfig indicates the EdgeDNS config
type EdgeDNSConfig struct {
	// Enable indicates whether enable EdgeDNS
	// default false
	Enable bool `json:"enable,omitempty"`
	// KubeAPIConfig is equivalent to EdgeMeshAgentConfig.KubeAPIConfig
	// do not allow users to configure manually
	KubeAPIConfig *KubeAPIConfig
	// ListenInterface indicates the listen interface of EdgeDNS
	// do not allow users to configure manually
	ListenInterface string
	// ListenPort indicates the listen port of EdgeDNS
	// default 53
	ListenPort int `json:"listenPort,omitempty"`
	// CacheDNS indicates the node local cache dns
	CacheDNS *CacheDNS `json:"cacheDNS,omitempty"`
}

type CacheDNS struct {
	// Enable indicates whether enable node local cache dns
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

// EdgeProxyConfig indicates the EdgeProxy config
type EdgeProxyConfig struct {
	// Enable indicates whether enable EdgeProxy
	// default false
	Enable bool `json:"enable,omitempty"`
	// ListenInterface indicates the listen interface of EdgeProxy
	// do not allow users to configure manually
	ListenInterface string
	// Socks5Proxy indicates the socks5 proxy config
	Socks5Proxy *Socks5Proxy `json:"socks5Proxy,omitempty"`
	// LoadBalancer indicates the load balance strategy
	LoadBalancer *LoadBalancer `json:"loadBalancer,omitempty"`
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
	// do not allow users to configure manually
	NodeName string
	// Namespace indicates namespace of host
	// do not allow users to configure manually
	Namespace string
}

// EdgeGatewayConfig indicates the EdgeGateway config
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
	//// GoChassisConfig defines some configurations related to go-chassis
	//// +Required
	//GoChassisConfig *GoChassisConfig `json:"goChassisConfig,omitempty"`
	// LoadBalancer indicates the load balance strategy
	LoadBalancer *LoadBalancer `json:"loadBalancer,omitempty"`
}

// LoadBalancer indicates the loadbalance strategy in edgemesh
type LoadBalancer struct {
	// Caller indicates which module using LoadBalancer
	// do not allow users to configure manually
	// options: ProxyCaller, GatewayCaller
	Caller defaults.LoadBalancerCaller
	// NodeName indicates name of host
	// do not allow users to configure manually
	NodeName string
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
	// do not allow users to configure manually
	// options: ServerAndClient, ClientOnly
	Mode defaults.TunnelMode
	// NodeName indicates the node name of EdgeTunnel
	// do not allow users to configure manually
	NodeName string
	// ListenPort indicates the listen port of EdgeTunnel
	// default 20006
	ListenPort int `json:"listenPort,omitempty"`
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
	// PSK configures libp2p to use the given private network protector.
	PSK *PSK `json:"psk,omitempty"`
}

type RelayNode struct {
	// NodeName indicates the relay node name, which is the same as the node name of Kubernetes
	NodeName string `json:"nodeName,omitempty"`
	// AdvertiseAddress sets the IP address for the relay node to advertise
	AdvertiseAddress []string `json:"advertiseAddress,omitempty"`
}

type PSK struct {
	// Enable indicates whether libp2p pnet is enabled.
	// default true
	Enable bool `json:"enable,omitempty"`
	// Path indicates the psk file path.
	// default /etc/edgemesh/psk
	Path string `json:"path,omitempty"`
}
