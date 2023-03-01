package proxy

import (
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
)

func (proxy *EdgeProxy) Run() {
	// start sock5 proxy
	if proxy.Config.Socks5Proxy.Enable {
		go proxy.Socks5Proxy.Start(wait.NeverStop)
	}

	// run proxy server
	err := proxy.ProxyServer.Run()
	if err != nil {
		klog.Errorf("run proxy server err: %v", err)
		return
	}

	<-beehiveContext.Done()
}
