package config

import (
	"os"

	"k8s.io/klog/v2"

	"github.com/kubeedge/edgemesh/common/acl"
	meshConstants "github.com/kubeedge/edgemesh/common/constants"
)

type TunnelAgentConfig struct {
	// Enable indicates whether TunnelAgent is enabled,
	// if set to false (for debugging etc.), skip checking other TunnelAgent configs.
	// default true
	Enable bool `json:"enable"`
	// TunnelACLConfig indicates the set of tunnel agent config about acl
	acl.TunnelACLConfig
	// NodeName indicates the node name of tunnel agent
	NodeName string `json:"nodeName"`
	// ListenPort indicates the listen port of tunnel agent
	// default 20006
	ListenPort int `json:"listenPort"`
}

func NewTunnelAgentConfig() *TunnelAgentConfig {
	nodeName, isExist := os.LookupEnv(meshConstants.MY_NODE_NAME)
	if !isExist {
		klog.Fatalf("env %s not exist", meshConstants.MY_NODE_NAME)
	}

	return &TunnelAgentConfig{
		Enable: true,
		TunnelACLConfig: acl.TunnelACLConfig{
			TLSPrivateKeyFile: meshConstants.AgentDefaultKeyFile,
		},
		NodeName:   nodeName,
		ListenPort: 20006,
	}
}
