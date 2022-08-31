package config

import (
	meshConstants "github.com/kubeedge/edgemesh/common/constants"
	"github.com/kubeedge/edgemesh/common/security"
)

const (
	defaultListenPort = 20006
	defaultRendezvous = "EdgeMesh_PlayGround"
)

type EdgeTunnelConfig struct {
	// Enable indicates whether TunnelAgent is enabled,
	// if set to false (for debugging etc.), skip checking other TunnelAgent configs.
	// default false
	Enable bool `json:"enable,omitempty"`
	// Security indicates the set of tunnel agent config about security
	Security *security.Security `json:"security,omitempty"`
	// NodeName indicates the node name of tunnel agent
	NodeName string `json:"nodeName,omitempty"`
	// ListenPort indicates the listen port of tunnel agent
	// default 20006
	ListenPort int `json:"listenPort,omitempty"`
	// EnableHolePunch indicates whether p2p hole punching feature is enabled,
	// default true
	EnableHolePunch bool `json:"enableHolePunch,omitempty"`
	// Transport indicates the transport protocol used by the p2p tunnel
	// default tcp
	Transport string `json:"transport,omitempty"`
	// Rendezvous unique string to identify group of libp2p nodes
	// default EdgeMesh_PlayGround
	Rendezvous string `json:"rendezvous,omitempty"`
	// RelayNodes indicates some nodes that can become libp2p relay nodes
	RelayNodes []*RelayNode `json:"relayNodes,omitempty"`
	// EnableIpfsLog open ipfs log info
	// default false
	EnableIpfsLog bool `json:"enableIpfsLog,omitempty"`
}

type RelayNode struct {
	// NodeName indicates the relay node name, which is the same as the node name of Kubernetes
	NodeName string `json:"nodeName,omitempty"`
	// AdvertiseAddress sets the IP address for the relay node to advertise
	AdvertiseAddress []string `json:"advertiseAddress,omitempty"`
}

func NewEdgeTunnelConfig() *EdgeTunnelConfig {
	return &EdgeTunnelConfig{
		Enable: false,
		Security: &security.Security{
			Enable:            false,
			TLSPrivateKeyFile: meshConstants.AgentDefaultKeyFile,
			TLSCAFile:         meshConstants.AgentDefaultCAFile,
			TLSCertFile:       meshConstants.AgentDefaultCertFile,
		},
		ListenPort:      defaultListenPort,
		EnableHolePunch: true,
		Transport:       "tcp",
		Rendezvous:      defaultRendezvous,
		EnableIpfsLog:   false,
	}
}
