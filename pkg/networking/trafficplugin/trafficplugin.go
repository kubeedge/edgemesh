package trafficplugin

import (
	"github.com/go-chassis/go-archaius"
	"github.com/go-chassis/go-chassis/control"
	"github.com/go-chassis/go-chassis/core/config"
	"github.com/go-chassis/go-chassis/core/config/model"
	"github.com/go-chassis/go-chassis/core/loadbalancer"
	"github.com/go-chassis/go-chassis/core/registry"
	"k8s.io/klog/v2"

	pluginConfig "github.com/kubeedge/edgemesh/pkg/networking/trafficplugin/config"
	"github.com/kubeedge/edgemesh/pkg/networking/trafficplugin/loadbalancer/consistenthash"
	_ "github.com/kubeedge/edgemesh/pkg/networking/trafficplugin/panel"
	pluginRegistry "github.com/kubeedge/edgemesh/pkg/networking/trafficplugin/registry"
)

// Install installs go-chassis plugins
func Install() {
	// service discovery
	opt := registry.Options{}
	registry.DefaultServiceDiscoveryService = pluginRegistry.NewEdgeServiceDiscovery(opt)
	// load balance
	for _, strategy := range pluginConfig.Config.LoadBalancer.SupportedLBStrategies {
		switch strategy {
		case loadbalancer.StrategyRoundRobin:
			loadbalancer.InstallStrategy(strategy, func() loadbalancer.Strategy {
				return &loadbalancer.RoundRobinStrategy{}
			})
		case loadbalancer.StrategyRandom:
			loadbalancer.InstallStrategy(strategy, func() loadbalancer.Strategy {
				return &loadbalancer.RandomStrategy{}
			})
		case consistenthash.StrategyConsistentHash:
			loadbalancer.InstallStrategy(strategy, func() loadbalancer.Strategy {
				return &consistenthash.Strategy{}
			})
		default:
			klog.Warningf("unsupported strategy name: %s", strategy)
		}
	}
	// control panel
	config.GlobalDefinition = &model.GlobalCfg{
		Panel: model.ControlPanel{
			Infra: "edge",
		},
		Ssl: make(map[string]string),
	}
	opts := control.Options{
		Infra:   config.GlobalDefinition.Panel.Infra,
		Address: config.GlobalDefinition.Panel.Settings["address"],
	}
	if err := control.Init(opts); err != nil {
		klog.Errorf("failed to init control: %v", err)
	}
	// init archaius
	if err := archaius.Init(); err != nil {
		klog.Errorf("failed to init archaius: %v", err)
	}
}
