package tcp

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/go-chassis/go-chassis/core/handler"
	"github.com/go-chassis/go-chassis/core/invocation"
	"k8s.io/klog/v2"

	"github.com/kubeedge/edgemesh/agent/pkg/chassis/config"
	"github.com/kubeedge/edgemesh/agent/pkg/tunnel"
	"github.com/kubeedge/edgemesh/agent/pkg/tunnel/proxy"
	"github.com/kubeedge/edgemesh/common/util"
)

const (
	l4ProxyHandlerName = "l4Proxy"
)

type TCPPROTO string

// L4ProxyHandler l4 proxy handler
type L4ProxyHandler struct{}

// Name name
func (h *L4ProxyHandler) Name() string {
	return l4ProxyHandlerName
}

// Handle handle
func (h *L4ProxyHandler) Handle(chain *handler.Chain, i *invocation.Invocation, cb invocation.ResponseCallBack) {
	r := &invocation.Response{}

	tcpProtocol, ok := i.Ctx.Value(TCPPROTO("tcp")).(*TCP)
	if !ok {
		r.Err = fmt.Errorf("can not get lconn from context")
		return
	}
	lconn := tcpProtocol.Conn

	epSplit := strings.Split(i.Endpoint, ":")
	if len(epSplit) != 3 {
		r.Err = fmt.Errorf("endpoint %s not a valid address", i.Endpoint)
		return
	}

	targetNodeName := epSplit[0]
	targetIP := epSplit[1]
	targetPort, err := strconv.ParseInt(epSplit[2], 10, 32)
	if err != nil {
		r.Err = fmt.Errorf("endpoint %s not a valid address", i.Endpoint)
		return
	}
	klog.Infof("l4 proxy get tcpserver address: %v", i.Endpoint)

	if targetNodeName == "" || targetNodeName == config.Chassis.Protocol.NodeName {
		addr := &net.TCPAddr{
			IP:   net.ParseIP(targetIP),
			Port: int(targetPort),
		}
		var rconn net.Conn
		defaultTCPReconnectTimes := config.Chassis.Protocol.TCPReconnectTimes
		defaultTCPClientTimeout := time.Second * time.Duration(config.Chassis.Protocol.TCPClientTimeout)
		for retry := 0; retry < defaultTCPReconnectTimes; retry++ {
			rconn, err = net.DialTimeout("tcp", addr.String(), defaultTCPClientTimeout)
			if err == nil {
				break
			}
		}
		if err != nil {
			r.Err = fmt.Errorf("l4 proxy dial error: %v", err)
			return
		}

		if tcpProtocol.UpgradeReq != nil {
			_, err = rconn.Write(tcpProtocol.UpgradeReq)
			if err != nil {
				r.Err = fmt.Errorf("tcp write req err: %s", err)
				return
			}
		}

		klog.Infof("l4 proxy start proxy data between tcpserver %s", addr.String())

		go util.ProxyConn(rconn, lconn)

		klog.Infof("Success proxy to %v", i.Endpoint)
		err = cb(r)
		if err != nil {
			klog.Warningf("Callback err: %v", err)
		}
	} else {
		proxyOpts := proxy.ProxyOptions{
			Protocol: "tcp",
			NodeName: targetNodeName,
			IP:       targetIP,
			Port:     int32(targetPort),
		}
		stream, err := tunnel.Agent.ProxySvc.GetProxyStream(proxyOpts)
		if err != nil {
			r.Err = fmt.Errorf("l4 proxy get proxy stream from %s error: %v", targetNodeName, err)
			return
		}
		klog.Infof("l4 proxy start proxy data between tcpserver %v", i.Endpoint)

		if tcpProtocol.UpgradeReq != nil {
			_, err = stream.Write(tcpProtocol.UpgradeReq)
			if err != nil {
				r.Err = fmt.Errorf("tcp write req err: %v", err)
				return
			}
		}

		go util.ProxyConn(stream, lconn)

		klog.Infof("Success proxy to %v", i.Endpoint)
		err = cb(r)
		if err != nil {
			klog.Warningf("Callback err: %v", err)
		}
	}
}

func newL4ProxyHandler() handler.Handler {
	return &L4ProxyHandler{}
}
