package dns

import (
	"fmt"
	"net"

	mdns "github.com/miekg/dns"

	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/edgemesh/agent/pkg/dns/config"
	"github.com/kubeedge/edgemesh/agent/pkg/dns/controller"
	"github.com/kubeedge/edgemesh/common/informers"
	"github.com/kubeedge/edgemesh/common/modules"
	"github.com/kubeedge/edgemesh/common/util"
)

// EdgeDNS is a node-level dns resolver
type EdgeDNS struct {
	Config   *config.EdgeDNSConfig
	ListenIP net.IP
	Server   *mdns.Server
}

func newEdgeDNS(c *config.EdgeDNSConfig, ifm *informers.Manager) (dns *EdgeDNS, err error) {
	dns = &EdgeDNS{Config: c}
	if !c.Enable {
		return dns, nil
	}

	// init dns controller
	controller.Init(ifm)

	// get dns listen ip
	dns.ListenIP, err = util.GetInterfaceIP(dns.Config.ListenInterface)
	if err != nil {
		return dns, fmt.Errorf("get dns listen ip err: %v", err)
	}

	addr := fmt.Sprintf("%v:%v", dns.ListenIP, dns.Config.ListenPort)
	dns.Server = &mdns.Server{Addr: addr, Net: "udp"}

	return dns, nil
}

// Register register edgedns to beehive modules
func Register(c *config.EdgeDNSConfig, ifm *informers.Manager) error {
	dns, err := newEdgeDNS(c, ifm)
	if err != nil {
		return fmt.Errorf("register module edgedns error: %v", err)
	}
	core.Register(dns)
	return nil
}

// Name of edgedns
func (dns *EdgeDNS) Name() string {
	return modules.EdgeDNSModuleName
}

// Group of edgedns
func (dns *EdgeDNS) Group() string {
	return modules.EdgeDNSModuleName
}

// Enable indicates whether enable this module
func (dns *EdgeDNS) Enable() bool {
	return dns.Config.Enable
}

// Start edgedns
func (dns *EdgeDNS) Start() {
	dns.Run()
}
