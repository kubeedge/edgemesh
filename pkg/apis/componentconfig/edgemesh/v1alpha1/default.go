package v1alpha1

import (
	"path"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeedge/edgemesh/pkg/common/constants"
)

// NewDefaultEdgeMeshConfig returns a full EdgeMeshConfig object
func NewDefaultEdgeMeshConfig() *EdgeMeshConfig {
	e := &EdgeMeshConfig{
		TypeMeta: metav1.TypeMeta{
			Kind:       Kind,
			APIVersion: path.Join(GroupName, APIVersion),
		},
		KubeAPIConfig: &KubeAPIConfig{
			Master:      "127.0.0.1:10550",
			ContentType: constants.DefaultKubeContentType,
			QPS:         constants.DefaultKubeQPS,
			Burst:       constants.DefaultKubeBurst,
			KubeConfig:  "",
		},
		Modules: &Modules{
			Networking: &Networking{
				Enable: true,
				TrafficPlugin: &TrafficPlugin{
					Enable: true,
					Protocol: &Protocol{
						TCPBufferSize:     8129,
						TCPClientTimeout:  2,
						TCPReconnectTimes: 3,
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
				},
				ServiceDiscovery: &ServiceDiscovery{
					Enable:          false,
					SubNet:          "10.0.0.0/24",
					ListenInterface: "docker0",
					ListenPort:      40001,
				},
				EdgeGateway: &EdgeGateway{
					Enable: false,
					NIC:    "*",
				},
			},
			Controller: &Controller{
				Enable: true,
				Buffer: &ControllerBuffer{
					ServiceEvent:         constants.DefaultServiceEventBuffer,
					EndpointsEvent:       constants.DefaultEndpointsEventBuffer,
					DestinationRuleEvent: constants.DefaultDestinationRuleEventBuffer,
					GatewayEvent:         constants.DefaultGatewayEventBuffer,
				},
			},
		},
	}
	return e
}
