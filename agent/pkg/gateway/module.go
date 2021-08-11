package gateway

import (
	"fmt"

	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/edgemesh/agent/pkg/gateway/config"
	"github.com/kubeedge/edgemesh/agent/pkg/gateway/controller"
	"github.com/kubeedge/edgemesh/common/informers"
	"github.com/kubeedge/edgemesh/common/modules"
)

// EdgeGateway is a edge ingress gateway
type EdgeGateway struct {
	Config *config.EdgeGatewayConfig
}

func newEdgeGateway(c *config.EdgeGatewayConfig, ifm *informers.Manager) (gw *EdgeGateway, err error) {
	gw = &EdgeGateway{Config: c}
	if !c.Enable {
		return gw, nil
	}

	// new controller
	controller.Init(ifm, c)

	return gw, nil
}

// Register register edgegateway to beehive modules
func Register(c *config.EdgeGatewayConfig, ifm *informers.Manager) error {
	gw, err := newEdgeGateway(c, ifm)
	if err != nil {
		return fmt.Errorf("register module edgegateway error: %v", err)
	}
	core.Register(gw)
	return nil
}

// Name of edgegateway
func (gw *EdgeGateway) Name() string {
	return modules.EdgeGatewayModuleName
}

// Group of edgegateway
func (gw *EdgeGateway) Group() string {
	return modules.EdgeGatewayModuleName
}

// Enable indicates whether enable this module
func (gw *EdgeGateway) Enable() bool {
	return gw.Config.Enable
}

// Start edgegateway
func (gw *EdgeGateway) Start() {
}
