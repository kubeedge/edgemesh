package manager

import (
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"

	"github.com/kubeedge/edgemesh/pkg/controller/config"
)

// ServiceManager manage all events of servicediscovery by SharedInformer
type ServiceManager struct {
	events chan watch.Event
}

// Events return the channel save events from watch servicediscovery change
func (sm *ServiceManager) Events() chan watch.Event {
	return sm.events
}

// NewServiceManager create ServiceManager by kube clientset and namespace
func NewServiceManager(si cache.SharedIndexInformer) (*ServiceManager, error) {
	events := make(chan watch.Event, config.Config.Buffer.ServiceEvent)
	rh := NewCommonResourceEventHandler(events)
	si.AddEventHandler(rh)

	return &ServiceManager{events: events}, nil
}
