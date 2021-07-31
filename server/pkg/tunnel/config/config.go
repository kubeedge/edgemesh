package config

import (
	"sync"

	"github.com/kubeedge/edgemesh/pkg/apis/componentconfig/edgemesh-server/v1alpha1"
)

var Config Configure
var once sync.Once

type Configure struct {
	v1alpha1.Tunnel
	NodeName string
}

func InitConfigure(tunnel *v1alpha1.Tunnel) {
	once.Do(func() {
		Config = Configure{
			Tunnel:   *tunnel,
			NodeName: tunnel.HostnameOverride,
		}
	})
}