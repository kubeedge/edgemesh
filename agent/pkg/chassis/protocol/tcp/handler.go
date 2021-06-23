package tcp

import (
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chassis/go-chassis/core/handler"
	"github.com/go-chassis/go-chassis/core/invocation"
	"k8s.io/klog/v2"
)

const (
	l4ProxyHandlerName = "l4Proxy"
)

// L4ProxyHandler l4 proxy handler
type L4ProxyHandler struct{}

// Name name
func (h *L4ProxyHandler) Name() string {
	return l4ProxyHandlerName
}

// Handle handle
func (h *L4ProxyHandler) Handle(chain *handler.Chain, i *invocation.Invocation, cb invocation.ResponseCallBack) {
	r := &invocation.Response{}

	tcpProtocol, ok := i.Ctx.Value("tcp").(TCP)
	if !ok {
		r.Err = fmt.Errorf("can not get lconn from context")
		return
	}
	lconn := tcpProtocol.Conn

	epSplit := strings.Split(i.Endpoint, ":")
	if len(epSplit) != 2 {
		r.Err = fmt.Errorf("endpoint %s not a valid address", i.Endpoint)
		return
	}

	host := epSplit[0]
	port, err := strconv.Atoi(epSplit[1])
	if err != nil {
		r.Err = fmt.Errorf("endpoint %s not a valid address", i.Endpoint)
		return
	}

	addr := &net.TCPAddr{
		IP:   net.ParseIP(host),
		Port: port,
	}
	klog.Infof("l4 proxy get httpserver address: %v", addr)
	var rconn net.Conn
	defaultTCPReconnectTimes := config.Config.Protocol.TCPReconnectTimes
	defaultTCPClientTimeout := time.Second * time.Duration(config.Config.Protocol.TCPClientTimeout)
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
	defer rconn.Close()

	if tcpProtocol.UpgradeReq != nil {
		_, err = rconn.Write(tcpProtocol.UpgradeReq)
		if err != nil {
			r.Err = fmt.Errorf("tcp write req err: %s", err)
			return
		}
	}

	klog.Infof("l4 proxy start a proxy to httpserver %s", addr.String())

	// TODO use context timeout ?
	wg := sync.WaitGroup{}
	wg.Add(2)
	go pipe(lconn, rconn, &wg)
	go pipe(rconn, lconn, &wg)

	wg.Wait()
	cb(r)
}

// 这里要了解下读或者写结束的时候，会返回什么结束码
func pipe(src, des io.ReadWriteCloser, wg *sync.WaitGroup) {
	// TODO 如何处理中断
	_, err := io.Copy(des, src)
	if err != nil {
		fmt.Println("read error: ", err)
	}
	wg.Done()
}


func newL4ProxyHandler() handler.Handler {
	return &L4ProxyHandler{}
}