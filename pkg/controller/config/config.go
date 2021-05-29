package config

import (
	"sync"

	"github.com/kubeedge/edgemesh/pkg/apis/componentconfig/edgemesh/v1alpha1"
)

var Config Configure
var once sync.Once

type Configure struct {
	v1alpha1.Controller
}

func InitConfigure(c *v1alpha1.Controller) {
	once.Do(func() {
		Config = Configure{
			Controller: *c,
		}
	})
}
