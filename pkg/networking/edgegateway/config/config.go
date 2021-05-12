package config

import (
	"sync"

	"github.com/kubeedge/edgemesh/pkg/apis/componentconfig/edgemesh/v1alpha1"
)

var Config Configure
var once sync.Once

type Configure struct {
	v1alpha1.EdgeGateway
}

func InitConfigure(e *v1alpha1.EdgeGateway) {
	once.Do(func() {
		Config = Configure{
			EdgeGateway: *e,
		}
	})
}
