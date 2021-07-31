package config

import (
	"os"

	"github.com/kubeedge/edgemesh/common/acl"
	meshConstants "github.com/kubeedge/edgemesh/common/constants"
	"k8s.io/klog/v2"
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
	// default 10006
	ListenPort int `json:"listenPort"`
}

func NewTunnelAgentConfig() *TunnelAgentConfig {
	nodeName, isExist := os.LookupEnv(meshConstants.MY_NODE_NAME)
	if !isExist {
		klog.Fatalf("env %s not exist", meshConstants.MY_NODE_NAME)
		os.Exit(1)
	}

	return &TunnelAgentConfig{
		Enable: true,
		TunnelACLConfig: acl.TunnelACLConfig{
			TLSPrivateKeyFile:  meshConstants.AgentDefaultKeyFile,
		},
		NodeName:   nodeName,
		ListenPort: 10006,
	}
}
