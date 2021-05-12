package networking

import (
	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/edgemesh/pkg/apis/componentconfig/edgemesh/v1alpha1"
	"github.com/kubeedge/edgemesh/pkg/common/modules"
	"github.com/kubeedge/edgemesh/pkg/networking/edgegateway"
	gatewayConfig "github.com/kubeedge/edgemesh/pkg/networking/edgegateway/config"
	"github.com/kubeedge/edgemesh/pkg/networking/servicediscovery"
	discoveryConfig "github.com/kubeedge/edgemesh/pkg/networking/servicediscovery/config"
	"github.com/kubeedge/edgemesh/pkg/networking/trafficplugin"
	pluginConfig "github.com/kubeedge/edgemesh/pkg/networking/trafficplugin/config"
)

type Networking struct {
	enable bool
}

func newNetworking(enable bool) *Networking {
	return &Networking{enable: enable}
}

func Register(n *v1alpha1.Networking) {
	pluginConfig.InitConfigure(n.TrafficPlugin)
	discoveryConfig.InitConfigure(n.ServiceDiscovery)
	gatewayConfig.InitConfigure(n.EdgeGateway)
	core.Register(newNetworking(n.Enable))
}

// Name of Networking
func (n *Networking) Name() string {
	return modules.NetworkingModuleName
}

// Group of Networking
func (n *Networking) Group() string {
	return modules.NetworkingGroupName
}

// Enable indicates whether enable this module
func (n *Networking) Enable() bool {
	return n.enable
}

// Start Networking
func (n *Networking) Start() {
	if pluginConfig.Config.Enable {
		trafficplugin.Install()
	}
	if discoveryConfig.Config.Enable {
		servicediscovery.Init()
		servicediscovery.Start()
	}
	if gatewayConfig.Config.Enable {
		edgegateway.Init()
	}
	klog.Infof("start networking process")
	for {
		select {
		case <-beehiveContext.Done():
			klog.Warning("EdgeMesh Stop")
			if discoveryConfig.Config.Enable {
				servicediscovery.Stop()
			}
			return
		default:
		}
		msg, err := beehiveContext.Receive(modules.NetworkingModuleName)
		if err != nil {
			klog.Warningf("[EdgeMesh] receive msg error %v", err)
			continue
		}
		klog.V(4).Infof("[EdgeMesh] get message: %v", msg)
		process(msg)
	}
}
