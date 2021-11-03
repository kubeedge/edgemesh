package tcp

import (
	"context"
	"fmt"
	"net"

	"github.com/go-chassis/go-chassis/core/common"
	"github.com/go-chassis/go-chassis/core/handler"
	"github.com/go-chassis/go-chassis/core/invocation"
	"k8s.io/klog/v2"

	"github.com/kubeedge/edgemesh/agent/pkg/chassis/loadbalancer/util"
)

func init() {
	err := handler.RegisterHandler(l4ProxyHandlerName, newL4ProxyHandler)
	if err != nil {
		klog.Errorf("register l4 proxy handler err: %v", err)
	}
}

// TCP tcp
type TCP struct {
	Conn         net.Conn
	SvcNamespace string
	SvcName      string
	Port         int
	// for websocket
	UpgradeReq []byte
}

// Process process
func (p *TCP) Process() {
	// create invocation
	inv := invocation.New(context.WithValue(context.Background(), TCPPROTO("tcp"), p))

	// set invocation
	inv.MicroServiceName = fmt.Sprintf("%s.%s.svc.cluster.local:%d", p.SvcName, p.SvcNamespace, p.Port)
	inv.SourceServiceID = ""
	inv.Protocol = "tcp"
	inv.Strategy = util.GetStrategyName(p.SvcNamespace, p.SvcName)
	inv.Args = p.UpgradeReq

	// create handlerchain
	c, err := handler.CreateChain(common.Consumer, "tcp", handler.Loadbalance, l4ProxyHandlerName)
	if err != nil {
		klog.Errorf("create handler chain error: %v", err)
		return
	}

	// start to handle
	c.Next(inv, p.responseCallback)
}

// responseCallback process invocation response
func (p *TCP) responseCallback(data *invocation.Response) error {
	if data.Err != nil {
		klog.Errorf("handle l4 proxy err : %v", data.Err)
		err := p.Conn.Close()
		if err != nil {
			klog.Errorf("close conn err: %v", err)
		}
		return data.Err
	}
	return nil
}
