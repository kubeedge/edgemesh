package manager

import (
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"

	"github.com/kubeedge/edgemesh/pkg/controller/config"
)

// DestinationRuleManager manage all events of destination rule by SharedInformer
type DestinationRuleManager struct {
	events chan watch.Event
}

// Events return the channel save events from watch destination rule change
func (sm *DestinationRuleManager) Events() chan watch.Event {
	return sm.events
}

// NewDestinationRuleManager create DestinationRuleManager by kube clientset and namespace
func NewDestinationRuleManager(si cache.SharedIndexInformer) (*DestinationRuleManager, error) {
	events := make(chan watch.Event, config.Config.Buffer.DestinationRuleEvent)
	rh := NewCommonResourceEventHandler(events)
	si.AddEventHandler(rh)

	return &DestinationRuleManager{events: events}, nil
}
