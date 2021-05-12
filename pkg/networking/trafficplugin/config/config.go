package config

import (
	"sync"

	"github.com/kubeedge/edgemesh/pkg/apis/componentconfig/edgemesh/v1alpha1"
)

var Config Configure
var once sync.Once

type Configure struct {
	v1alpha1.TrafficPlugin
}

func InitConfigure(p *v1alpha1.TrafficPlugin) {
	once.Do(func() {
		Config = Configure{
			TrafficPlugin: *p,
		}
	})
}
