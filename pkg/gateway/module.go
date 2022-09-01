package gateway

import (
	"fmt"
	"github.com/kubeedge/edgemesh/pkg/apis/config/defaults"

	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/edgemesh/pkg/apis/config/v1alpha1"
	"github.com/kubeedge/edgemesh/pkg/common/informers"
	"github.com/kubeedge/edgemesh/pkg/gateway/chassis"
	"github.com/kubeedge/edgemesh/pkg/gateway/controller"
	"github.com/kubeedge/edgemesh/pkg/gateway/manager"
)

// EdgeGateway is a edge ingress gateway
type EdgeGateway struct {
	Config *v1alpha1.EdgeGatewayConfig
}

// Name of edgegateway
func (gw *EdgeGateway) Name() string {
	return defaults.EdgeGatewayModuleName
}

// Group of edgegateway
func (gw *EdgeGateway) Group() string {
	return defaults.EdgeGatewayModuleName
}

// Enable indicates whether enable this module
func (gw *EdgeGateway) Enable() bool {
	return gw.Config.Enable
}

// Start edgegateway
func (gw *EdgeGateway) Start() {
}

// Register register edgegateway to beehive modules
func Register(c *v1alpha1.EdgeGatewayConfig, ifm *informers.Manager) error {
	gw, err := newEdgeGateway(c, ifm)
	if err != nil {
		return fmt.Errorf("register module %s error: %v", defaults.EdgeGatewayModuleName, err)
	}
	core.Register(gw)
	return nil
}

func newEdgeGateway(c *v1alpha1.EdgeGatewayConfig, ifm *informers.Manager) (gw *EdgeGateway, err error) {
	gw = &EdgeGateway{Config: c}
	if !c.Enable {
		return gw, nil
	}

	chassis.Install(c.GoChassisConfig, ifm)

	// new controller
	controller.Init(ifm, c)

	// new gateway manager
	manager.NewGatewayManager(c)

	return gw, nil
}
