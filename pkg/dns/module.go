package dns

import (
	"fmt"

	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/edgemesh/pkg/apis/config/defaults"
	"github.com/kubeedge/edgemesh/pkg/apis/config/v1alpha1"
	"github.com/kubeedge/edgemesh/pkg/clients"
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

// Shutdown EdgeDNS
func (d *EdgeDNS) Shutdown() {
	// TODO graceful shutdown
}

// Register edgedns to beehive modules
func Register(c *v1alpha1.EdgeDNSConfig, cli *clients.Clients) error {
	dns, err := newEdgeDNS(c, cli)
	if err != nil {
		return fmt.Errorf("register module edgedns error: %v", err)
	}
	core.Register(dns)
	return nil
}

func newEdgeDNS(c *v1alpha1.EdgeDNSConfig, cli *clients.Clients) (*EdgeDNS, error) {
	if !c.Enable {
		return &EdgeDNS{Config: c}, nil
	}

	// update Corefile for node-local dns
	err := UpdateCorefile(c, cli.GetKubeClient())
	if err != nil {
		return nil, fmt.Errorf("failed to update corefile, err: %w", err)
	}

	return &EdgeDNS{Config: c}, nil
}
