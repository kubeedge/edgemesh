package servicediscovery

import (
	"github.com/kubeedge/edgemesh/pkg/networking/servicediscovery/dns"
	"github.com/kubeedge/edgemesh/pkg/networking/servicediscovery/proxier"
	"github.com/kubeedge/edgemesh/pkg/networking/servicediscovery/serviceproxy"
)

// Init init
func Init() {
	// init tcp service proxy
	serviceproxy.Init()
	// init iptables
	proxier.Init()
	// init dns server
	dns.Init()
}

// Start starts all service discovery components
func Start() {
	go serviceproxy.StartServiceProxy()
	go proxier.StartProxier()
	go dns.StartDNS()
}

// Stop stop
func Stop() {
	proxier.Clean()
}
