package dns

import (
	"fmt"

	"github.com/coredns/coredns/coremain"
	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/edgemesh/agent/pkg/dns/config"
	"github.com/kubeedge/edgemesh/common/informers"
	"github.com/kubeedge/edgemesh/common/modules"
)

// EdgeDNS is a node-level dns resolver
type EdgeDNS struct {
	Config *config.EdgeDNSConfig
}

func newEdgeDNS(c *config.EdgeDNSConfig, ifm *informers.Manager) (dns *EdgeDNS, err error) {
	dns = &EdgeDNS{Config: c}
	if !c.Enable {
		return dns, nil
	}

	// update Corefile for node-local dns
	err = UpdateCorefile(c, ifm)
	if err != nil {
		return dns, fmt.Errorf("failed to update corefile, err: %w", err)
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
	if dns.Config.CacheDNS.Enable {
		klog.Infof("Runs CoreDNS v%s as a cache dns", coremain.CoreVersion)
	} else {
		klog.Infof("Runs CoreDNS v%s as a local dns", coremain.CoreVersion)
	}
	coremain.Run()
}
