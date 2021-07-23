package dns

import (
	"fmt"
	"net"

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
	DNSConn  *net.UDPConn
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

	laddr := &net.UDPAddr{
		IP:   dns.ListenIP,
		Port: dns.Config.ListenPort,
	}
	dns.DNSConn, err = net.ListenUDP("udp", laddr)
	if err != nil {
		return dns, fmt.Errorf("dns server listen on %v error: %v", laddr, err)
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
