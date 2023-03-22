package v1alpha1

import (
	"os"
	"path"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/kubeedge/edgemesh/pkg/apis/config/defaults"
	"github.com/kubeedge/kubeedge/common/constants"
)

var defaultKubeConfig = &KubeAPIConfig{
	Master:      "",
	ContentType: runtime.ContentTypeProtobuf,
	QPS:         constants.DefaultKubeQPS,
	Burst:       constants.DefaultKubeBurst,
	KubeConfig:  "",
	Mode:        defaults.ManualMode,
	MetaServer: &MetaServer{
		Server: defaults.MetaServerAddress,
		Security: &MetaServerSecurity{
			RequireAuthorization:  false,
			InsecureSkipTLSVerify: false,
			TLSCaFile:             defaults.MetaServerCaFile,
			TLSCertFile:           defaults.MetaServerCertFile,
			TLSPrivateKeyFile:     defaults.MetaServerKeyFile,
		},
	},
	DeleteKubeConfig: false,
}

var defaultLoadBalancerConfig = &LoadBalancer{
	Caller: defaults.ProxyCaller,
	ConsistentHash: &ConsistentHash{
		PartitionCount:    100,
		ReplicationFactor: 10,
		Load:              1.25,
	},
}

func newDefaultEdgeTunnelConfig(configPath string) *EdgeTunnelConfig {
	return &EdgeTunnelConfig{
		Enable:          false,
		Mode:            defaults.ServerClientMode,
		ListenPort:      20006,
		Transport:       "tcp",
		Rendezvous:      defaults.Rendezvous,
		EnableIpfsLog:   false,
		MaxCandidates:   15,
		HeartbeatPeriod: 120,
		FinderPeriod:    60,
		PSK: &PSK{
			Enable: true,
			Path:   defaults.PSKPath,
		},
		TunnelLimitConfig: &TunnelLimitConfig{
			Enable:                  true,
			TunnelBaseStreamIn:      defaults.TunnelBaseStreamIn,
			TunnelBaseStreamOut:     defaults.TunnelBaseStreamOut,
			TunnelPeerBaseStreamIn:  defaults.TunnelPeerBaseStreamIn,
			TunnelPeerBaseStreamOut: defaults.TunnelPeerBaseStreamOut,
		},
		ConfigPath:       configPath,
		ListenInterfaces: "*",
	}
}

// NewDefaultEdgeMeshAgentConfig returns a full EdgeMeshAgentConfig object
func NewDefaultEdgeMeshAgentConfig(configPath string) *EdgeMeshAgentConfig {
	c := &EdgeMeshAgentConfig{
		TypeMeta: metav1.TypeMeta{
			Kind:       "EdgeMeshAgent",
			APIVersion: path.Join("agent.edgemesh.config.kubeedge.io", "v1alpha1"),
		},
		KubeAPIConfig: defaultKubeConfig,
		CommonConfig: &CommonConfig{
			BridgeDeviceName: defaults.BridgeDeviceName,
			BridgeDeviceIP:   defaults.BridgeDeviceIP,
		},
		Modules: &AgentModules{
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
				Enable:            false,
				ServiceFilterMode: defaults.FilterIfLabelExistsMode,
				Socks5Proxy: &Socks5Proxy{
					Enable:     false,
					ListenPort: 10800,
				},
				LoadBalancer: defaultLoadBalancerConfig,
			},
			EdgeTunnelConfig: newDefaultEdgeTunnelConfig(configPath),
		},
	}

	preConfigAgent(c, DetectRunningMode())
	return c
}

// NewDefaultEdgeMeshGatewayConfig returns a full EdgeMeshGatewayConfig object
func NewDefaultEdgeMeshGatewayConfig(configPath string) *EdgeMeshGatewayConfig {
	c := &EdgeMeshGatewayConfig{
		TypeMeta: metav1.TypeMeta{
			Kind:       "EdgeMeshGateway",
			APIVersion: path.Join("gateway.edgemesh.config.kubeedge.io", "v1alpha1"),
		},
		KubeAPIConfig: defaultKubeConfig,
		Modules: &GatewayModules{
			EdgeGatewayConfig: &EdgeGatewayConfig{
				Enable:       false,
				NIC:          "*",
				IncludeIP:    "*",
				ExcludeIP:    "*",
				LoadBalancer: defaultLoadBalancerConfig,
			},
			EdgeTunnelConfig: newDefaultEdgeTunnelConfig(configPath),
		},
	}

	preConfigGateway(c, DetectRunningMode())
	return c
}

// DetectRunningMode detects whether the container is running on cloud node or edge node.
// It will recognize whether there is KUBERNETES_PORT in the container environment variable, because
// edged will not inject KUBERNETES_PORT environment variable into the container, but kubelet will.
// What is edged: https://kubeedge.io/en/docs/architecture/edge/edged/
func DetectRunningMode() defaults.RunningMode {
	_, exist := os.LookupEnv("KUBERNETES_PORT")
	if exist {
		return defaults.CloudMode
	}
	return defaults.EdgeMode
}

// preConfigAgent will init the edgemesh-agent configuration according to the mode.
func preConfigAgent(c *EdgeMeshAgentConfig, mode defaults.RunningMode) {
	c.KubeAPIConfig.Mode = mode

	if mode == defaults.EdgeMode {
		// ContentType only supports application/json
		// see issue: https://github.com/kubeedge/kubeedge/issues/3041
		c.KubeAPIConfig.ContentType = runtime.ContentTypeJSON
		// when edgemesh-agent is running on the edge, we enable the EdgeDNS module by default.
		// EdgeDNS replaces CoreDNS or kube-dns to respond to domain name requests from edge applications.
		c.Modules.EdgeDNSConfig.Enable = true
	}

	if mode == defaults.CloudMode {
		// when edgemesh-agent is running on the cloud, we do not need to enable EdgeDNS,
		// because all dns request can be done by coredns or kube-dns.
		c.Modules.EdgeDNSConfig.Enable = false
	}
}

// preConfigGateway will init the edgemesh-gateway configuration according to the mode.
func preConfigGateway(c *EdgeMeshGatewayConfig, mode defaults.RunningMode) {
	c.KubeAPIConfig.Mode = mode

	if mode == defaults.EdgeMode {
		c.KubeAPIConfig.ContentType = runtime.ContentTypeJSON
	}
}
