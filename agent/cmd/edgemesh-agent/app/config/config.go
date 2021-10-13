package config

import (
	"io/ioutil"
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
	DefaultConfigMapName   = "edgemesh-agent-cfg"
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
	// DummyDeviceName indicates the name of the dummy device will be created
	// default edgemesh0
	DummyDeviceName string `json:"dummyDeviceName,omitempty"`
	// DummyDeviceIP indicates the IP bound to the dummy device
	// default "169.254.96.16"
	DummyDeviceIP string `json:"dummyDeviceIP,omitempty"`
	// ConfigMapName indicates the configmap mounted by edgemesh-agent,
	// which contains all the configuration information of edgemesh-agent
	// default edgemesh-agent-cfg
	ConfigMapName string `json:"configMapName,omitempty"`
}

// Modules indicates the modules of edgeMeshAgent will be use
type Modules struct {
	// EdgeDNSConfig indicates edgedns module config
	EdgeDNSConfig *dnsconfig.EdgeDNSConfig `json:"edgeDNS,omitempty"`
	// EdgeProxyConfig indicates edgeproxy module config
	EdgeProxyConfig *proxyconfig.EdgeProxyConfig `json:"edgeProxy,omitempty"`
	// EdgeGatewayConfig indicates edgegateway module config
	EdgeGatewayConfig *gwconfig.EdgeGatewayConfig `json:"edgeGateway,omitempty"`
	// TunnelAgentConfig indicates tunnelagent module config
	TunnelAgentConfig *tunnelconfig.TunnelAgentConfig `json:"tunnel,omitempty"`
}

// NewEdgeMeshAgentConfig returns a full EdgeMeshAgentConfig object
func NewEdgeMeshAgentConfig() *EdgeMeshAgentConfig {
	c := &EdgeMeshAgentConfig{
		TypeMeta: metav1.TypeMeta{
			Kind:       Kind,
			APIVersion: path.Join(GroupName, APIVersion),
		},
		CommonConfig: &CommonConfig{
			DummyDeviceName: DefaultDummyDeviceName,
			DummyDeviceIP:   DefaultDummyDeviceIP,
			ConfigMapName:   DefaultConfigMapName,
		},
		KubeAPIConfig: &v1alpha1.KubeAPIConfig{
			Master:      "",
			ContentType: runtime.ContentTypeJSON,
			QPS:         constants.DefaultKubeQPS,
			Burst:       constants.DefaultKubeBurst,
			KubeConfig:  "",
		},
		GoChassisConfig: chassisconfig.NewGoChassisConfig(),
		Modules: &Modules{
			EdgeDNSConfig:     dnsconfig.NewEdgeDNSConfig(),
			EdgeProxyConfig:   proxyconfig.NewEdgeProxyConfig(),
			EdgeGatewayConfig: gwconfig.NewEdgeGatewayConfig(),
			TunnelAgentConfig: tunnelconfig.NewTunnelAgentConfig(),
		},
	}

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
