package config

import (
	"sync"

	"github.com/kubeedge/edgemesh/pkg/apis/config/v1alpha1"
)

var (
	once    sync.Once
	Chassis Configure
)

type Configure struct {
	v1alpha1.GoChassisConfig
}

// InitConfigure init go-chassis configures
func InitConfigure(c *v1alpha1.GoChassisConfig) {
	once.Do(func() {
		Chassis = Configure{
			GoChassisConfig: *c,
		}
	})
}
