package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EdgeMeshConfig indicates the config of edgeMesh which get from edgeMesh config file
type EdgeMeshConfig struct {
	metav1.TypeMeta
	// KubeAPIConfig indicates the kubernetes cluster info which cloudCore will connected
	// +Required
	KubeAPIConfig *KubeAPIConfig `json:"kubeAPIConfig,omitempty"`
	// Modules indicates edgeMesh modules config
	// +Required
	Modules *Modules `json:"modules,omitempty"`
}

// KubeAPIConfig indicates the configuration for interacting with k8s server
type KubeAPIConfig struct {
	// Master indicates the address of the Kubernetes API server (overrides any value in KubeConfig)
	// such as https://127.0.0.1:8443
	// default ""
	// Note: Can not use "omitempty" option,  It will affect the output of the default configuration file
	Master string `json:"master"`
	// ContentType indicates the ContentType of message transmission when interacting with k8s
	// default "application/vnd.kubernetes.protobuf"
	ContentType string `json:"contentType,omitempty"`
	// QPS to while talking with kubernetes apiserve
	// default 100
	QPS int32 `json:"qps,omitempty"`
	// Burst to use while talking with kubernetes apiserver
	// default 200
	Burst int32 `json:"burst,omitempty"`
	// KubeConfig indicates the path to kubeConfig file with authorization and master location information.
	// default "/root/.kube/config"
	// +Required
	KubeConfig string `json:"kubeConfig"`
}

// Modules indicates the modules of EdgeMesh will be use
type Modules struct {
	// Networking indicates networking module config
	Networking *Networking `json:"networking,omitempty"`
	// Controller indicates controller module config
	Controller *Controller `json:"controller,omitempty"`
}

// Networking indicates networking module config
type Networking struct {
	// Enable indicates whether Networking is enabled,
	// if set to false (for debugging etc.), skip checking other Networking configs.
	// default true
	Enable bool `json:"enable,omitempty"`
	// Plugin indicates the go-chassis plugins traffic config for Networking module
	// Optional if traffic plugin is configured
	TrafficPlugin *TrafficPlugin `json:"trafficPlugin,omitempty"`
	// ServiceDiscovery indicates service discovery config for Networking module
	// Optional if service discovery is configured
	ServiceDiscovery *ServiceDiscovery `json:"serviceDiscovery,omitempty"`
	// EdgeGateway indicates edge gateway config for Networking module
	// Optional if edge gateway is configured
	EdgeGateway *EdgeGateway `json:"edgeGateway,omitempty"`
}

// TrafficPlugin indicates the go-chassis traffic plugins config
type TrafficPlugin struct {
	// Enable indicates whether enable go-chassis plugins
	// default true
	Enable bool `json:"enable,omitempty"`
	// Protocol indicates the network protocols config supported in the traffic plugin
	Protocol *Protocol `json:"protocol,omitempty"`
	// LoadBalancer indicates the load balance strategy
	LoadBalancer *LoadBalancer `json:"loadBalancer,omitempty"`
}

// Protocol indicates the network protocols config supported in the traffic plugin
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
}

// LoadBalancer indicates the load balance strategy in the traffic plugin
type LoadBalancer struct {
	// DefaultLBStrategy indicates default load balance strategy name
	// default "RoundRobin"
	DefaultLBStrategy string `json:"defaultLBStrategy,omitempty"`
	// SupportedLBStrategies indicates supported load balance strategies name
	// default []string{"RoundRobin", "Random", "ConsistentHash"}
	SupportedLBStrategies []string `json:"SupportLBStrategies,omitempty"`
	// ConsistentHash indicates the extension of the go-chassis loadbalancer
	ConsistentHash *ConsistentHash `json:"consistentHash,omitempty"`
}

// ConsistentHash strategy is an extension of the go-chassis loadbalancer
// For more information about the underlying algorithm, please take a look at
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

// ServiceDiscovery indicates the service discovery config
type ServiceDiscovery struct {
	// Enable indicates whether enable service discovery
	// default true
	Enable bool `json:"enable,omitempty"`
	// SubNet indicates the subnet of proxier
	// default "10.0.0.0/24", equals to k8s default service-cluster-ip-range
	SubNet string `json:"subNet,omitempty"`
	// ListenInterface indicates the listen interface of EdgeMesh
	// default "docker0"
	ListenInterface string `json:"listenInterface,omitempty"`
	// ListenPort indicates the listen port of EdgeMesh
	// default 40001
	ListenPort int `json:"listenPort,omitempty"`
}

// EdgeGateway indicates the edge gateway config
type EdgeGateway struct {
	// Enable indicates whether enable edge gateway
	// default true
	Enable bool `json:"enable,omitempty"`
	// NIC indicates the network interface controller that the edge gateway needs to listen to.
	// empty or "*" stands for all netcards. You can also specify network adapters such as "lo,eth0"
	// default "*"
	NIC string `json:"nic,omitempty"`
	// IncludeIP indicates the host IP that the edge gateway needs to listen to.
	// empty or "*" stands for all ips. You can also specify ips such as "192.168.1.56,10.5.2.1"
	// default ""
	IncludeIP string `json:"includeIP,omitempty"`
	// ExcludeIP indicates the IP address that the edge gateway does not want to listen to.
	// empty or "*" stands for not exclude any ip. You can also specify ips such as "192.168.1.56,10.5.2.1"
	// default ""
	ExcludeIP string `json:"excludeIP,omitempty"`
}

// Controller indicates the config of controller module
type Controller struct {
	// Enable indicates whether Controller is enabled,
	// if set to false (for debugging etc.), skip checking other Controller configs.
	// default true
	Enable bool `json:"enable,omitempty"`
	// Buffer indicates k8s resource buffer
	Buffer *ControllerBuffer `json:"buffer,omitempty"`
}

// ControllerBuffer indicates the Controller buffer
type ControllerBuffer struct {
	// UpdatePodStatus indicates the buffer of pod status
	// default 1024
	UpdatePodStatus int32 `json:"updatePodStatus,omitempty"`
	// UpdateNodeStatus indicates the buffer of update node status
	// default 1024
	UpdateNodeStatus int32 `json:"updateNodeStatus,omitempty"`
	// QueryConfigMap indicates the buffer of query configMap
	// default 1024
	QueryConfigMap int32 `json:"queryConfigMap,omitempty"`
	// QuerySecret indicates the buffer of query secret
	// default 1024
	QuerySecret int32 `json:"querySecret,omitempty"`
	// QueryService indicates the buffer of query service
	// default 1024
	QueryService int32 `json:"queryService,omitempty"`
	// QueryEndpoints indicates the buffer of query endpoint
	// default 1024
	QueryEndpoints int32 `json:"queryEndpoints,omitempty"`
	// PodEvent indicates the buffer of pod event
	// default 1
	PodEvent int32 `json:"podEvent,omitempty"`
	// ConfigMapEvent indicates the buffer of configMap event
	// default 1
	ConfigMapEvent int32 `json:"configMapEvent,omitempty"`
	// SecretEvent indicates the buffer of secret event
	// default 1
	SecretEvent int32 `json:"secretEvent,omitempty"`
	// ServiceEvent indicates the buffer of service event
	// default 1
	ServiceEvent int32 `json:"serviceEvent,omitempty"`
	// EndpointsEvent indicates the buffer of endpoint event
	// default 1
	EndpointsEvent int32 `json:"endpointsEvent,omitempty"`
	// RulesEvent indicates the buffer of rule event
	// default 1
	RulesEvent int32 `json:"rulesEvent,omitempty"`
	// RuleEndpointsEvent indicates the buffer of endpoint event
	// default 1
	RuleEndpointsEvent int32 `json:"ruleEndpointsEvent,omitempty"`
	// DestinationRuleEvent indicates the buffer of destination rule event
	// default 1
	DestinationRuleEvent int32 `json:"destinationRuleEvent,omitempty"`
	// GatewayEvent indicates the buffer of gateway event
	// default 1
	GatewayEvent int32 `json:"gatewayEvent,omitempty"`
	// QueryPersistentVolume indicates the buffer of query persistent volume
	// default 1024
	QueryPersistentVolume int32 `json:"queryPersistentVolume,omitempty"`
	// QueryPersistentVolumeClaim indicates the buffer of query persistent volume claim
	// default 1024
	QueryPersistentVolumeClaim int32 `json:"queryPersistentVolumeClaim,omitempty"`
	// QueryVolumeAttachment indicates the buffer of query volume attachment
	// default 1024
	QueryVolumeAttachment int32 `json:"queryVolumeAttachment,omitempty"`
	// QueryNode indicates the buffer of query node
	// default 1024
	QueryNode int32 `json:"queryNode,omitempty"`
	// UpdateNode indicates the buffer of update node
	// default 1024
	UpdateNode int32 `json:"updateNode,omitempty"`
	// DeletePod indicates the buffer of delete pod message from edge
	// default 1024
	DeletePod int32 `json:"deletePod,omitempty"`
}
