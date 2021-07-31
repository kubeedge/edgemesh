package tcp

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/go-chassis/go-chassis/core/common"
	"github.com/go-chassis/go-chassis/core/handler"
	"github.com/go-chassis/go-chassis/core/invocation"
	"k8s.io/klog/v2"

	"github.com/kubeedge/edgemesh/agent/pkg/chassis/config"
	"github.com/kubeedge/edgemesh/agent/pkg/chassis/loadbalancer/util"
)

const l4ProxyHandlerName = "l4Proxy"

// L4ProxyHandler l4 proxy handler
type L4ProxyHandler struct{}

// Name name
func (h *L4ProxyHandler) Name() string {
	return l4ProxyHandlerName
}

// Handle handle
func (h *L4ProxyHandler) Handle(chain *handler.Chain, i *invocation.Invocation, cb invocation.ResponseCallBack) {
	r := &invocation.Response{
		Result: i.Endpoint,
	}
	if err := cb(r); err != nil {
		fmt.Println(fmt.Errorf("failed to cb: %s", err.Error()))
	}
}

func newL4ProxyHandler() handler.Handler {
	return &L4ProxyHandler{}
}

func init() {
	err := handler.RegisterHandler(l4ProxyHandlerName, newL4ProxyHandler)
	if err != nil {
		klog.Errorf("register l4 proxy handler err: %v", err)
	}
}

type conntrack struct {
	lconn net.Conn
	rconn net.Conn
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
	inv := invocation.New(context.Background())

	// set invocation
	inv.MicroServiceName = fmt.Sprintf("%s.%s.svc.cluster.local:%d", p.SvcName, p.SvcNamespace, p.Port)
	inv.SourceServiceID = ""
	if p.UpgradeReq == nil {
		inv.Protocol = "tcp"
	} else {
		// websocket
		inv.Protocol = "rest"
	}
	inv.Strategy = util.GetStrategyName(p.SvcNamespace, p.SvcName)
	inv.Args = p.UpgradeReq

	// create handlerchain
	c, err := handler.CreateChain(common.Consumer, "tcp", handler.Loadbalance, l4ProxyHandlerName)
	if err != nil {
		klog.Errorf("create handler chain error: %v", err)
		err = p.Conn.Close()
		if err != nil {
			klog.Errorf("close conn err: %v", err)
		}
		return
	}

	// start to handle
	c.Next(inv, p.responseCallback)
}

// responseCallback process invocation response
func (p *TCP) responseCallback(data *invocation.Response) error {
	if data.Err != nil {
		klog.Errorf("handle l4 proxy err: %v", data.Err)
		err := p.Conn.Close()
		if err != nil {
			klog.Errorf("close conn err: %v", err)
		}
		return data.Err
	}

	ep, ok := data.Result.(string)
	if !ok {
		klog.Errorf("result %v not string type", data.Result)
		err := p.Conn.Close()
		if err != nil {
			klog.Errorf("close conn err: %v", err)
		}
		return fmt.Errorf("result %v not string type", data.Result)
	}
	epSplit := strings.Split(ep, ":")
	if len(epSplit) != 2 {
		klog.Errorf("endpoint %s not a valid address", ep)
		err := p.Conn.Close()
		if err != nil {
			klog.Errorf("close conn err: %v", err)
		}
		return fmt.Errorf("endpoint %s not a valid address", ep)
	}
	host := epSplit[0]
	port, err := strconv.Atoi(epSplit[1])
	if err != nil {
		klog.Errorf("endpoint %s not a valid address", ep)
		err1 := p.Conn.Close()
		if err1 != nil {
			klog.Errorf("close conn err: %v", err1)
		}
		return fmt.Errorf("endpoint %s not a valid address", ep)
	}
	addr := &net.TCPAddr{
		IP:   net.ParseIP(host),
		Port: port,
	}
	klog.Infof("l4 proxy get httpserver address: %v", addr)
	var proxyClient net.Conn
	defaultTCPReconnectTimes := config.Chassis.Protocol.TCPReconnectTimes
	defaultTCPClientTimeout := time.Second * time.Duration(config.Chassis.Protocol.TCPClientTimeout)
	for retry := 0; retry < defaultTCPReconnectTimes; retry++ {
		proxyClient, err = net.DialTimeout("tcp", addr.String(), defaultTCPClientTimeout)
		if err == nil {
			break
		}
	}
	// error when connecting to httpserver, maybe timeout or any other error
	if err != nil {
		klog.Errorf("l4 proxy dial httpserver error: %v", err)
		err = p.Conn.Close()
		if err != nil {
			klog.Errorf("close conn err: %v", err)
		}
		return err
	}

	ctk := &conntrack{
		lconn: p.Conn,
		rconn: proxyClient,
	}

	// do websocket req
	if p.UpgradeReq != nil {
		_, err = ctk.rconn.Write(p.UpgradeReq)
		if err != nil {
			klog.Errorf("tcp write req err: %s", err)
			err = p.Conn.Close()
			if err != nil {
				klog.Errorf("close conn err: %v", err)
			}
			err = ctk.rconn.Close()
			if err != nil {
				klog.Errorf("close conn err: %v", err)
			}
			return err
		}
		klog.Infof("write websocket upgrade req success")
	}

	klog.Infof("l4 Proxy start a proxy to httpserver %s", addr.String())
	go ctk.processServerProxy()
	go ctk.processClientProxy()
	return nil
}

// processServerProxy process up link traffic
func (c *conntrack) processClientProxy() {
	buf := make([]byte, config.Chassis.Protocol.TCPBufferSize)
	for {
		n, err := c.lconn.Read(buf)
		// service caller closes the connection
		if n == 0 {
			err = c.lconn.Close()
			if err != nil {
				klog.Errorf("lconn close err: %v", err)
			}
			err = c.rconn.Close()
			if err != nil {
				klog.Errorf("rconn close err: %v", err)
			}
			break
		}
		if err != nil {
			klog.Infof("processClientProxy read err: %s", err)
			err = c.lconn.Close()
			if err != nil {
				klog.Errorf("lconn close err: %v", err)
			}
			err = c.rconn.Close()
			if err != nil {
				klog.Errorf("rconn close err: %v", err)
			}
			break
		}
		_, err = c.rconn.Write(buf[:n])
		if err != nil {
			klog.Infof("processClientProxy write err: %s", err)
			err = c.lconn.Close()
			if err != nil {
				klog.Errorf("lconn close err: %v", err)
			}
			err = c.rconn.Close()
			if err != nil {
				klog.Errorf("rconn close err: %v", err)
			}
			break
		}
	}
}

// processServerProxy process down link traffic
func (c *conntrack) processServerProxy() {
	buf := make([]byte, config.Chassis.Protocol.TCPBufferSize)
	for {
		n, err := c.rconn.Read(buf)
		if n == 0 {
			err = c.rconn.Close()
			if err != nil {
				klog.Errorf("rconn close err: %v", err)
			}
			err = c.lconn.Close()
			if err != nil {
				klog.Errorf("lconn close err: %v", err)
			}
			break
		}
		if err != nil {
			klog.Infof("processServerProxy read err: %s", err)
			err = c.rconn.Close()
			if err != nil {
				klog.Errorf("rconn close err: %v", err)
			}
			err = c.lconn.Close()
			if err != nil {
				klog.Errorf("lconn close err: %v", err)
			}
			break
		}
		_, err = c.lconn.Write(buf[:n])
		if err != nil {
			klog.Infof("processServerProxy write err: %s", err)
			err = c.rconn.Close()
			if err != nil {
				klog.Errorf("rconn close err: %v", err)
			}
			err = c.lconn.Close()
			if err != nil {
				klog.Errorf("lconn close err: %v", err)
			}
			break
		}
	}
}
