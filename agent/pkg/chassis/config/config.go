package config

import (
	meshConstants "github.com/kubeedge/edgemesh/common/constants"
	"k8s.io/klog/v2"
	"os"
	"sync"
)

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
	NodeName string `json:"nodeName"`
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

func NewGoChassisConfig() *GoChassisConfig {
	nodeName, isExist := os.LookupEnv(meshConstants.MY_NODE_NAME)
	if !isExist {
		klog.Fatalf("env %s not exist", meshConstants.MY_NODE_NAME)
		os.Exit(1)
	}

	return &GoChassisConfig{
		Protocol: &Protocol{
			TCPBufferSize:     8192,
			TCPClientTimeout:  2,
			TCPReconnectTimes: 3,
			NodeName: nodeName,
		},
		LoadBalancer: &LoadBalancer{
			DefaultLBStrategy:     "RoundRobin",
			SupportedLBStrategies: []string{"RoundRobin", "Random", "ConsistentHash"},
			ConsistentHash: &ConsistentHash{
				PartitionCount:    100,
				ReplicationFactor: 10,
				Load:              1.25,
			},
		},
	}
}

var (
	once    sync.Once
	Chassis Configure
)

type Configure struct {
	GoChassisConfig
}

// InitConfigure init go-chassis configures
func InitConfigure(c *GoChassisConfig) {
	once.Do(func() {
		Chassis = Configure{
			GoChassisConfig: *c,
		}
	})
}
