package config

import (
	"io/ioutil"
	"os"
	"path"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	"sigs.k8s.io/yaml"

	chassisconfig "github.com/kubeedge/edgemesh/agent/pkg/chassis/config"
	dnsconfig "github.com/kubeedge/edgemesh/agent/pkg/dns/config"
	gwconfig "github.com/kubeedge/edgemesh/agent/pkg/gateway/config"
	proxyconfig "github.com/kubeedge/edgemesh/agent/pkg/proxy/config"
	tunnelconfig "github.com/kubeedge/edgemesh/agent/pkg/tunnel/config"
	"github.com/kubeedge/kubeedge/common/constants"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/cloudcore/v1alpha1"
)

const (
	GroupName  = "agent.edgemesh.config.kubeedge.io"
	APIVersion = "v1alpha1"
	Kind       = "EdgeMeshAgent"

	DefaultDummyDeviceName = "edgemesh0"
	DefaultDummyDeviceIP   = "169.254.96.16"
	DefaultEdgeApiServer   = "http://127.0.0.1:10550"

	// EdgeMode means that edgemesh-agent detects that it is currently running on the edge
	EdgeMode = "EdgeMode"
	// CloudMode means that edgemesh-agent detects that it is currently running on the cloud
	CloudMode = "CloudMode"
	// DebugMode indicates that the user manually configured kubeAPIConfig
	DebugMode = "DebugMode"
)

// EdgeMeshAgentConfig indicates the config of edgeMeshAgent which get from edgeMeshAgent config file
type EdgeMeshAgentConfig struct {
	metav1.TypeMeta
	// CommonConfig indicates common config for all modules
	// +Required
	CommonConfig *CommonConfig `json:"commonConfig,omitempty"`
	// KubeAPIConfig indicates the kubernetes cluster info which edgeMeshAgent will connected
	// +Required
	KubeAPIConfig *v1alpha1.KubeAPIConfig `json:"kubeAPIConfig,omitempty"`
	// GoChassisConfig defines some configurations related to go-chassis
	// +Required
	GoChassisConfig *chassisconfig.GoChassisConfig `json:"goChassisConfig,omitempty"`
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
	EdgeDNSConfig *dnsconfig.EdgeDNSConfig `json:"edgeDNS,omitempty"`
	// EdgeProxyConfig indicates edgeproxy module config
	EdgeProxyConfig *proxyconfig.EdgeProxyConfig `json:"edgeProxy,omitempty"`
	// EdgeGatewayConfig indicates edgegateway module config
	EdgeGatewayConfig *gwconfig.EdgeGatewayConfig `json:"edgeGateway,omitempty"`
	// EdgeTunnelConfig indicates tunnel module config
	EdgeTunnelConfig *tunnelconfig.EdgeTunnelConfig `json:"tunnel,omitempty"`
}

// NewEdgeMeshAgentConfig returns a full EdgeMeshAgentConfig object
func NewEdgeMeshAgentConfig() *EdgeMeshAgentConfig {
	c := &EdgeMeshAgentConfig{
		TypeMeta: metav1.TypeMeta{
			Kind:       Kind,
			APIVersion: path.Join(GroupName, APIVersion),
		},
		CommonConfig: &CommonConfig{
			Mode:            DebugMode,
			DummyDeviceName: DefaultDummyDeviceName,
			DummyDeviceIP:   DefaultDummyDeviceIP,
		},
		KubeAPIConfig: &v1alpha1.KubeAPIConfig{
			Master:      "",
			ContentType: runtime.ContentTypeProtobuf,
			QPS:         constants.DefaultKubeQPS,
			Burst:       constants.DefaultKubeBurst,
			KubeConfig:  "",
		},
		GoChassisConfig: chassisconfig.NewGoChassisConfig(),
		Modules: &Modules{
			EdgeDNSConfig:     dnsconfig.NewEdgeDNSConfig(),
			EdgeProxyConfig:   proxyconfig.NewEdgeProxyConfig(),
			EdgeGatewayConfig: gwconfig.NewEdgeGatewayConfig(),
			EdgeTunnelConfig:  tunnelconfig.NewEdgeTunnelConfig(),
		},
	}

	preConfigByMode(c, detectRunningMode())
	return c
}

// Parse unmarshal config file into *EdgeMeshAgentConfig
func (c *EdgeMeshAgentConfig) Parse(filename string) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		klog.Errorf("Failed to read config file %s: %v", filename, err)
		return err
	}
	err = yaml.Unmarshal(data, c)
	if err != nil {
		klog.Errorf("Failed to unmarshal config file %s: %v", filename, err)
		return err
	}
	return nil
}

// detectRunningMode detects whether the edgemesh-agent is running on cloud node or edge node.
// It will recognize whether there is KUBERNETES_PORT in the container environment variable, because
// edged will not inject KUBERNETES_PORT environment variable into the container, but kubelet will.
// what is edged: https://kubeedge.io/en/docs/architecture/edge/edged/
func detectRunningMode() string {
	_, exist := os.LookupEnv("KUBERNETES_PORT")
	if exist {
		return CloudMode
	}
	return EdgeMode
}

// preConfigByMode will init the edgemesh-agent configuration according to the mode.
func preConfigByMode(c *EdgeMeshAgentConfig, mode string) {
	c.CommonConfig.Mode = mode

	if mode == EdgeMode {
		// edgemesh-agent relies on the local apiserver function of KubeEdge when it runs at the edge node.
		// KubeEdge v1.6+ starts to support this function until KubeEdge v1.7+ tends to be stable.
		// what is KubeEdge local apiserver: https://github.com/kubeedge/kubeedge/blob/master/CHANGELOG/CHANGELOG-1.6.md
		c.KubeAPIConfig.Master = DefaultEdgeApiServer
		// ContentType only supports application/json
		// see issue: https://github.com/kubeedge/kubeedge/issues/3041
		c.KubeAPIConfig.ContentType = runtime.ContentTypeJSON
		// when edgemesh-agent is running on the edge, we enable the edgedns module by default.
		// edgedns replaces CoreDNS or kube-dns to respond to domain name requests from edge applications.
		c.Modules.EdgeDNSConfig.Enable = true
	}

	if mode == CloudMode {
		c.KubeAPIConfig.Master = ""
		c.KubeAPIConfig.ContentType = runtime.ContentTypeProtobuf
		// when edgemesh-agent is running on the cloud, we do not need to enable edgedns,
		// because all domain name resolution can be done by CoreDNS or kube-dns.
		c.Modules.EdgeDNSConfig.Enable = false
	}
}
