package proxy

import (
	"k8s.io/klog/v2"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
)

func (proxy *EdgeProxy) Run() {
	// start sock5 proxy
	if proxy.Config.Socks5Proxy.Enable {
		go proxy.Socks5Proxy.Start()
	}

	// run proxy server
	err := proxy.ProxyServer.Run()
	if err != nil {
		klog.Errorf("run proxy server err: %v", err)
		return
	}

	// TODO graceful shutdown
	<-beehiveContext.Done()
	err = proxy.ProxyServer.CleanupAndExit()
	if err != nil {
		klog.ErrorS(err, "Cleanup iptables failed")
	}
}
