package v1alpha1

import (
	"os"
	"path"

	"github.com/kubeedge/kubeedge/common/constants"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/cloudcore/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/kubeedge/edgemesh/pkg/apis/config/defaults"
)

const (
	GroupName  = "agent.edgemesh.config.kubeedge.io"
	APIVersion = "v1alpha1"
	Kind       = "EdgeMeshAgent"
)

// NewDefaultEdgeMeshAgentConfig returns a full EdgeMeshAgentConfig object
func NewDefaultEdgeMeshAgentConfig() *EdgeMeshAgentConfig {
	c := &EdgeMeshAgentConfig{
		TypeMeta: metav1.TypeMeta{
			Kind:       Kind,
			APIVersion: path.Join(GroupName, APIVersion),
		},
		CommonConfig: &CommonConfig{
			Mode:            defaults.DebugMode,
			DummyDeviceName: defaults.DummyDeviceName,
			DummyDeviceIP:   defaults.DummyDeviceIP,
		},
		KubeAPIConfig: &v1alpha1.KubeAPIConfig{
			Master:      "",
			ContentType: runtime.ContentTypeProtobuf,
			QPS:         constants.DefaultKubeQPS,
			Burst:       constants.DefaultKubeBurst,
			KubeConfig:  "",
		},
		Modules: &Modules{
			EdgeDNSConfig: &EdgeDNSConfig{
				Enable:     false,
				ListenPort: 53,
				CacheDNS: &CacheDNS{
					Enable:     false,
					AutoDetect: true,
					CacheTTL:   30,
				},
			},
			EdgeProxyConfig: &EdgeProxyConfig{
				Enable: false,
				Socks5Proxy: &Socks5Proxy{
					Enable:     false,
					ListenPort: 10800,
				},
			},
			EdgeGatewayConfig: &EdgeGatewayConfig{
				Enable:    false,
				NIC:       "*",
				IncludeIP: "*",
				ExcludeIP: "*",
				GoChassisConfig: &GoChassisConfig{
					Protocol: &Protocol{
						TCPBufferSize:     8192,
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
			},
			EdgeTunnelConfig: &EdgeTunnelConfig{
				Enable:          false,
				ListenPort:      20006,
				EnableHolePunch: true,
				Transport:       "tcp",
				Rendezvous:      defaults.Rendezvous,
				EnableIpfsLog:   false,
				MaxCandidates:   5,
				HeartbeatPeriod: 120,
				FinderPeriod:    60,
				PSK: &PSK{
					Enable: true,
					Path:   defaults.PSKPath,
				},
			},
		},
	}

	preConfigByMode(c, detectRunningMode())
	return c
}

// detectRunningMode detects whether the edgemesh-agent is running on cloud node or edge node.
// It will recognize whether there is KUBERNETES_PORT in the container environment variable, because
// edged will not inject KUBERNETES_PORT environment variable into the container, but kubelet will.
// what is edged: https://kubeedge.io/en/docs/architecture/edge/edged/
func detectRunningMode() string {
	_, exist := os.LookupEnv("KUBERNETES_PORT")
	if exist {
		return defaults.CloudMode
	}
	return defaults.EdgeMode
}

// preConfigByMode will init the edgemesh-agent configuration according to the mode.
func preConfigByMode(c *EdgeMeshAgentConfig, mode string) {
	c.CommonConfig.Mode = mode

	if mode == defaults.EdgeMode {
		// edgemesh-agent relies on the local apiserver function of KubeEdge when it runs at the edge node.
		// KubeEdge v1.6+ starts to support this function until KubeEdge v1.7+ tends to be stable.
		// what is KubeEdge local apiserver: https://github.com/kubeedge/kubeedge/blob/master/CHANGELOG/CHANGELOG-1.6.md
		c.KubeAPIConfig.Master = defaults.MetaServerAddress
		// ContentType only supports application/json
		// see issue: https://github.com/kubeedge/kubeedge/issues/3041
		c.KubeAPIConfig.ContentType = runtime.ContentTypeJSON
		// when edgemesh-agent is running on the edge, we enable the edgedns module by default.
		// edgedns replaces CoreDNS or kube-dns to respond to domain name requests from edge applications.
		c.Modules.EdgeDNSConfig.Enable = true
	}

	if mode == defaults.CloudMode {
		c.KubeAPIConfig.Master = ""
		c.KubeAPIConfig.ContentType = runtime.ContentTypeProtobuf
		// when edgemesh-agent is running on the cloud, we do not need to enable edgedns,
		// because all domain name resolution can be done by CoreDNS or kube-dns.
		c.Modules.EdgeDNSConfig.Enable = false
	}
}
