package module

import (
	"fmt"
	"sync"

	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core"
)

var modules = make(map[string]module)

type module interface {
	core.Module
	Shutdown()
}

func Initialize(coreModules map[string]core.Module) error {
	for _, coreModule := range coreModules {
		m, ok := coreModule.(module)
		if !ok {
			return fmt.Errorf("can't convert %T to module", coreModule)
		}
		modules[m.Name()] = m
	}
	return nil
}

func Shutdown() {
	wg := sync.WaitGroup{}
	for _, m := range modules {
		wg.Add(1)
		go func(m module) {
			defer wg.Done()
			klog.Infof("Shutdown module %s", m.Name())
			m.Shutdown()
		}(m)
	}
	wg.Wait()
}
