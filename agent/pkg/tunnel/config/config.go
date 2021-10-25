package config

import (
	"os"

	"k8s.io/klog/v2"

	"github.com/kubeedge/edgemesh/common/acl"
	meshConstants "github.com/kubeedge/edgemesh/common/constants"
)

const (
	defaultListenPort = 20006
)

type TunnelAgentConfig struct {
	// Enable indicates whether TunnelAgent is enabled,
	// if set to false (for debugging etc.), skip checking other TunnelAgent configs.
	// default true
	Enable bool `json:"enable,omitempty"`
	// TunnelACLConfig indicates the set of tunnel agent config about acl
	TunnelACLConfig acl.TunnelACLConfig `json:"ACL,omitempty"`
	// NodeName indicates the node name of tunnel agent
	NodeName string `json:"nodeName,omitempty"`
	// ListenPort indicates the listen port of tunnel agent
	// default 20006
	ListenPort int `json:"listenPort,omitempty"`
	// EnableSecurity indicates whether to use the ca acl and security transport
	// default false
	EnableSecurity bool `json:"enableSecurity"`
}

func NewTunnelAgentConfig() *TunnelAgentConfig {
	nodeName, isExist := os.LookupEnv(meshConstants.MY_NODE_NAME)
	if !isExist {
		klog.Fatalf("env %s not exist", meshConstants.MY_NODE_NAME)
	}

	return &TunnelAgentConfig{
		Enable: false,
		TunnelACLConfig: acl.TunnelACLConfig{
			TLSPrivateKeyFile: meshConstants.AgentDefaultKeyFile,
			TLSCAFile:         meshConstants.AgentDefaultCAFile,
			TLSCertFile:       meshConstants.AgentDefaultCertFile,
		},
		NodeName:       nodeName,
		ListenPort:     defaultListenPort,
		EnableSecurity: false,
	}
}
