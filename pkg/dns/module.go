package dns

import (
	"fmt"

	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/edgemesh/pkg/apis/config/defaults"
	"github.com/kubeedge/edgemesh/pkg/apis/config/v1alpha1"
	"github.com/kubeedge/edgemesh/pkg/common/informers"
)

// EdgeDNS is a node-level dns resolver
type EdgeDNS struct {
	Config *v1alpha1.EdgeDNSConfig
}

// Name of EdgeDNS
func (d *EdgeDNS) Name() string {
	return defaults.EdgeDNSModuleName
}

// Group of EdgeDNS
func (d *EdgeDNS) Group() string {
	return defaults.EdgeDNSModuleName
}

// Enable indicates whether enable this module
func (d *EdgeDNS) Enable() bool {
	return d.Config.Enable
}

// Start EdgeDNS
func (d *EdgeDNS) Start() {
	d.Run()
}

// Register register edgedns to beehive modules
func Register(c *v1alpha1.EdgeDNSConfig, ifm *informers.Manager) error {
	dns, err := newEdgeDNS(c, ifm)
	if err != nil {
		return fmt.Errorf("register module edgedns error: %v", err)
	}
	core.Register(dns)
	return nil
}

func newEdgeDNS(c *v1alpha1.EdgeDNSConfig, ifm *informers.Manager) (dns *EdgeDNS, err error) {
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
