package config

import (
	"k8s.io/klog/v2"

	meshConstants "github.com/kubeedge/edgemesh/common/constants"
	"github.com/kubeedge/edgemesh/common/security"
	"github.com/kubeedge/edgemesh/common/util"
)

const (
	defaultListenPort = 20004
)

// TunnelServerConfig indicates networking module config
type TunnelServerConfig struct {
	// Enable indicates whether Tunnel is enabled,
	// if set to false (for debugging etc.), skip checking other Networking configs.
	// default true
	Enable bool `json:"enable,omitempty"`
	// Security indicates the set of tunnel server config about security
	Security *security.Security `json:"security,omitempty"`
	// NodeName indicates the node name of tunnel server
	NodeName string `json:"nodeName,omitempty"`
	// ListenPort indicates the listen port of tunnel server
	// default 20004
	ListenPort int `json:"listenPort,omitempty"`
	// AdvertiseAddress sets the IP address for the edgemesh-server to advertise
	AdvertiseAddress []string `json:"advertiseAddress,omitempty"`
	// Transport indicates the transport protocol used by the p2p tunnel
	Transport string `json:"transport,omitempty"`
}

func NewTunnelServerConfig() *TunnelServerConfig {
	// fetch the public IP auto and append it to the advertiseAddress
	publicIP := util.FetchPublicIP()
	advertiseAddress := make([]string, 0)
	if publicIP != "" {
		klog.Infof("Fetch public IP: %s", publicIP)
		advertiseAddress = append(advertiseAddress, publicIP)
	} else {
		klog.Infof("Unable to fetch public IP")
	}

	return &TunnelServerConfig{
		Enable: true,
		Security: &security.Security{
			Enable:            false,
			TLSPrivateKeyFile: meshConstants.ServerDefaultKeyFile,
			TLSCAFile:         meshConstants.ServerDefaultCAFile,
			TLSCertFile:       meshConstants.ServerDefaultCertFile,
		},
		ListenPort:       defaultListenPort,
		AdvertiseAddress: advertiseAddress,
		Transport:        "tcp",
	}
}
