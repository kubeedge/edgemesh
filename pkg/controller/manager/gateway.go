package manager

import (
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"

	"github.com/kubeedge/edgemesh/pkg/controller/config"
)

// GatewayManager manage all events of edgegateway by SharedInformer
type GatewayManager struct {
	events chan watch.Event
}

// Events return the channel save events from watch edgegateway change
func (gm *GatewayManager) Events() chan watch.Event {
	return gm.events
}

// NewGatewayManager create GatewayManager by kube clientset and namespace
func NewGatewayManager(si cache.SharedIndexInformer) (*GatewayManager, error) {
	events := make(chan watch.Event, config.Config.Buffer.GatewayEvent)
	rh := NewCommonResourceEventHandler(events)
	si.AddEventHandler(rh)

	return &GatewayManager{events: events}, nil
}
