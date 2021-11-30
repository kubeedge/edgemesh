package chassis

import (
	"github.com/go-chassis/go-archaius"
	"github.com/go-chassis/go-chassis/control"
	"github.com/go-chassis/go-chassis/core/config"
	"github.com/go-chassis/go-chassis/core/config/model"
	"github.com/go-chassis/go-chassis/core/loadbalancer"
	"github.com/go-chassis/go-chassis/core/registry"
	"k8s.io/klog/v2"

	chassisconfig "github.com/kubeedge/edgemesh/agent/pkg/chassis/config"
	"github.com/kubeedge/edgemesh/agent/pkg/chassis/controller"
	"github.com/kubeedge/edgemesh/agent/pkg/chassis/loadbalancer/consistenthash"
	_ "github.com/kubeedge/edgemesh/agent/pkg/chassis/panel"
	meshregistry "github.com/kubeedge/edgemesh/agent/pkg/chassis/registry"
	"github.com/kubeedge/edgemesh/common/informers"
	"github.com/kubeedge/edgemesh/common/util"
)

// Install installs go-chassis plugin
func Install(c *chassisconfig.GoChassisConfig, ifm *informers.Manager) {
	// init chassis configure
	c.Protocol.NodeName = util.FetchNodeName()
	chassisconfig.InitConfigure(c)
	// init chassis controller
	controller.Init(ifm)
	// service discovery
	opt := registry.Options{}
	registry.DefaultServiceDiscoveryService = meshregistry.NewEdgeServiceDiscovery(opt)
	// load balance
	for _, strategy := range c.LoadBalancer.SupportedLBStrategies {
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
