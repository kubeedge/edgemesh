package config

import (
	"k8s.io/klog/v2"
	"os"

	meshConstants "github.com/kubeedge/edgemesh/common/constants"
)

// EdgeProxyConfig indicates the edgeproxy config
type EdgeProxyConfig struct {
	// Enable indicates whether enable edgeproxy
	// default true
	Enable bool `json:"enable,omitempty"`
	// SubNet indicates the subnet of proxier, which equals to k8s service-cluster-ip-range
	SubNet string `json:"subNet,omitempty"`
	// ListenInterface indicates the listen interface of edgeproxy
	// do not allow users to configure manually
	ListenInterface string `json:"listenInterface,omitempty"`
	// ListenPort indicates the listen port of edgeproxy
	// default 40001
	ListenPort int `json:"listenPort,omitempty"`
	//default 10800
	SocksListenPort int `json:"socksListenPort,omitempty"`
	//host nodename
	NodeName string `json:"nodeName,omitempty"`
}

func NewEdgeProxyConfig() *EdgeProxyConfig {
	nodeName, isExist := os.LookupEnv(meshConstants.MY_NODE_NAME)
	if !isExist {
		klog.Fatalf("env %s not exist", meshConstants.MY_NODE_NAME)
	}

	return &EdgeProxyConfig{
		Enable:          false,
		ListenPort:      40001,
		SocksListenPort: 10800,
		NodeName:        nodeName,
	}
}
