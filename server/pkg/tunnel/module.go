package tunnel

import (
	"context"
	"fmt"

	"github.com/libp2p/go-libp2p"
	circuit "github.com/libp2p/go-libp2p-circuit"
	"github.com/libp2p/go-libp2p-core/host"
	ma "github.com/multiformats/go-multiaddr"
	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/edgemesh/common/acl"
	"github.com/kubeedge/edgemesh/common/informers"
	"github.com/kubeedge/edgemesh/common/modules"
	"github.com/kubeedge/edgemesh/server/pkg/tunnel/config"
	"github.com/kubeedge/edgemesh/server/pkg/tunnel/controller"
)

// TunnelServer is on cloud, as a signal and relay server
type TunnelServer struct {
	Config *config.TunnelServerConfig
	Host   host.Host
}

func newTunnelServer(c *config.TunnelServerConfig, ifm *informers.Manager) (server *TunnelServer, err error) {
	server = &TunnelServer{Config: c}
	if !c.Enable {
		return server, nil
	}

	controller.Init(ifm)

	privateKey, err := acl.GetPrivateKey(c.TunnelACLConfig)
	if err != nil {
		return server, err
	}

	var externalMultiAddr ma.Multiaddr
	if c.PublicIP != "" {
		externalMultiAddr, err = ma.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d", c.PublicIP, c.ListenPort))
		if err != nil {
			klog.Warningf("New multiaddr err: %v", err)
		}
	}
	addressFactory := func(addrs []ma.Multiaddr) []ma.Multiaddr {
		if externalMultiAddr != nil {
			addrs = append(addrs, externalMultiAddr)
		}
		return addrs
	}

	host, err := libp2p.New(
		context.Background(),
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", c.ListenPort)),
		libp2p.AddrsFactory(addressFactory),
		libp2p.EnableRelay(circuit.OptHop),
		libp2p.ForceReachabilityPrivate(),
		libp2p.Identity(privateKey),
	)
	if err != nil {
		errMsg := fmt.Errorf("Start tunnel server failed, %v", err)
		klog.Errorln(errMsg)
		return server, errMsg
	}
	server.Host = host

	return server, err
}

// Register register tunnelserver to beehive modules
func Register(c *config.TunnelServerConfig, ifm *informers.Manager) error {
	server, err := newTunnelServer(c, ifm)
	if err != nil {
		return fmt.Errorf("register module tunnelserver error: %v", err)
	}
	core.Register(server)
	return nil
}

// Name of tunnelserver
func (t *TunnelServer) Name() string {
	return modules.TunnelServerModuleName
}

// Group of tunnelserver
func (t *TunnelServer) Group() string {
	return modules.TunnelServerModuleName
}

// Enable indicates whether enable this module
func (t *TunnelServer) Enable() bool {
	return t.Config.Enable
}

// Start tunnelserver
func (t *TunnelServer) Start() {
	t.Run()
}
