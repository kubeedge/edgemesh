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
)

const (
	InterfaceAddress = "169.254.0.0/16"
	InterfaceName    = "edgemesh"
)

// EdgeDNS is a node-level dns resolver
type EdgeDNS struct {
	Config *config.EdgeDNSConfig
	Server *mdns.Server
}

func newEdgeDNS(c *config.EdgeDNSConfig, ifm *informers.Manager) (dns *EdgeDNS, err error) {
	dns = &EdgeDNS{Config: c}
	if !c.Enable {
		return dns, nil
	}

	// init dns controller
	controller.Init(ifm)

	dns.Server = &mdns.Server{
		Net:  "udp",
		Addr: fmt.Sprintf("%s:%d", net.ParseIP(InterfaceAddress).To4().String(), dns.Config.ListenPort),
	}

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
