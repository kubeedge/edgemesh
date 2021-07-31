package config

import (
	"io/ioutil"
	"os"
	"path"

	"github.com/kubeedge/edgemesh/common/acl"
	meshConstants "github.com/kubeedge/edgemesh/common/constants"
	tunnelserverconfig "github.com/kubeedge/edgemesh/server/pkg/tunnel/config"
	"github.com/kubeedge/kubeedge/common/constants"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/cloudcore/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	"sigs.k8s.io/yaml"
)

const (
	GroupName  = "server.edgemesh.config.kubeedge.io"
	APIVersion = "v1alpha1"
	Kind       = "EdgeMeshServer"
)

// EdgeMeshServerConfig indicates the config of edgeMeshServer which get from edgeMeshServer config file
type EdgeMeshServerConfig struct {
	metav1.TypeMeta
	// KubeAPIConfig indicates the kubernetes cluster info which cloudCore will connected
	// default use InClusterConfig
	// +Optional
	KubeAPIConfig *v1alpha1.KubeAPIConfig `json:"kubeAPIConfig,omitempty"`
	// Modules indicates edgeMesh modules config
	// +Required
	Modules *Modules `json:"modules,omitempty"`
}

// Modules indicates the modules of EdgeMesh-Server will be use
type Modules struct {
	// Tunnel indicates tunnel server module config
	TunnelServer *tunnelserverconfig.TunnelServerConfig `json:"tunnel,omitempty"`
}

// NewEdgeMeshServerConfig returns a full EdgeMeshServerConfig object
func NewEdgeMeshServerConfig() *EdgeMeshServerConfig {
	nodeName, isExist := os.LookupEnv(meshConstants.MY_NODE_NAME)
	if !isExist {
		klog.Fatalf("env %s not exist", meshConstants.MY_NODE_NAME)
		os.Exit(1)
	}

	c := &EdgeMeshServerConfig{
		TypeMeta: metav1.TypeMeta{
			Kind:       Kind,
			APIVersion: path.Join(GroupName, APIVersion),
		},
		KubeAPIConfig: &v1alpha1.KubeAPIConfig{
			Master:      "",
			ContentType: constants.DefaultKubeContentType,
			QPS:         constants.DefaultKubeQPS,
			Burst:       constants.DefaultKubeBurst,
			KubeConfig:  "",
		},
		Modules: &Modules{
			TunnelServer: &tunnelserverconfig.TunnelServerConfig{
				Enable: true,
				TunnelACLConfig: acl.TunnelACLConfig{
					TLSPrivateKeyFile:  meshConstants.ServerDefaultKeyFile,
				},
				NodeName:   nodeName,
				ListenPort: 10004,
				PublicIP:   "0.0.0.0",
			}},
	}
	return c
}

// Parse unmarshal config file into *EdgeMeshAgentConfig
func (c *EdgeMeshServerConfig) Parse(filename string) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		klog.Errorf("Failed to read configfile %s: %v", filename, err)
		return err
	}
	err = yaml.Unmarshal(data, c)
	if err != nil {
		klog.Errorf("Failed to unmarshal configfile %s: %v", filename, err)
		return err
	}
	return nil
}
