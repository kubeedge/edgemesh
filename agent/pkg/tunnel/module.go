package tunnel

import (
	"context"
	"fmt"
	"sync"
	"time"

	ipfslog "github.com/ipfs/go-log/v2"
	"github.com/kubeedge/beehive/pkg/core"
	"github.com/libp2p/go-libp2p"
	p2phost "github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/routing"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/p2p/host/autorelay"
	relayv1 "github.com/libp2p/go-libp2p/p2p/protocol/circuitv1/relay"
	ma "github.com/multiformats/go-multiaddr"
	"k8s.io/klog/v2"

	"github.com/kubeedge/edgemesh/agent/pkg/tunnel/config"
	discoverypb "github.com/kubeedge/edgemesh/agent/pkg/tunnel/pb/discovery"
	proxypb "github.com/kubeedge/edgemesh/agent/pkg/tunnel/pb/proxy"
	"github.com/kubeedge/edgemesh/common/informers"
	"github.com/kubeedge/edgemesh/common/modules"
	"github.com/kubeedge/edgemesh/common/util"
)

type TunnelMode string

const (
	ClientMode       TunnelMode = "ClientOnly"
	ServerClientMode TunnelMode = "ServerAndClient"
	UnknownMode      TunnelMode = "Unknown"

	defaultRendezvous = "edgemesh-rendezvous"
)

var Agent *EdgeTunnel

// EdgeTunnel is used for solving cross subset communication
type EdgeTunnel struct {
	Config *config.EdgeTunnelConfig

	p2pHost     p2phost.Host       // libp2p host
	hostCtx     context.Context    // ctx governs the lifetime of the libp2p host
	mu          sync.Mutex         // protect nodePeerMap
	nodePeerMap map[string]peer.ID // map of Kubernetes node name and peer.ID

	rendezvous   string // unique string to identify group of libp2p nodes
	mdnsPeerChan chan peer.AddrInfo
	dhtPeerChan  <-chan peer.AddrInfo

	isRelay      bool
	relayMaddrs  []ma.Multiaddr
	relayPeers   map[string]*peer.AddrInfo
	relayService *relayv1.Relay

	stopCh chan struct{}
}

func newEdgeTunnel(c *config.EdgeTunnelConfig, ifm *informers.Manager, mode TunnelMode) (*EdgeTunnel, error) {
	// for debug
	ipfslog.SetAllLoggers(ipfslog.LevelInfo)

	// TODO Set the NodeName variable in the outer function
	c.NodeName = util.FetchNodeName()

	ctx := context.Background()

	privKey, err := GenerateKeyPairWithString(c.NodeName)
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	var idht *dht.IpfsDHT
	opts := []libp2p.Option{
		libp2p.Identity(privKey),
		libp2p.ListenAddrStrings(GenerateMultiAddrString(c.Transport, "0.0.0.0", c.ListenPort)),
		libp2p.DefaultSecurity,
		GenerateTransportOption(c.Transport),
		libp2p.NATPortMap(),
		libp2p.Routing(func(h p2phost.Host) (routing.PeerRouting, error) {
			idht, err = dht.New(ctx, h)
			return idht, err
		}),
		libp2p.EnableAutoRelay(autorelay.WithCircuitV1Support(), autorelay.WithBootDelay(15*time.Second)),
		libp2p.EnableNATService(),
	}

	relayPeers, relayMaddrs := GenerateRelayRecord(c.RelayNodes, c.Transport, c.ListenPort)
	// If this host is a relay node, we need to append its advertiseAddress
	relayInfo, isRelay := relayPeers[c.NodeName]
	if isRelay {
		opts = append(opts, libp2p.AddrsFactory(func(maddrs []ma.Multiaddr) []ma.Multiaddr {
			maddrs = append(maddrs, relayInfo.Addrs...)
			return maddrs
		}))
	}

	if c.EnableHolePunch {
		opts = append(opts, libp2p.EnableHolePunching())
	}

	h, err := libp2p.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to new p2p host: %w", err)
	}
	klog.V(0).Infof("I'm %s\n", fmt.Sprintf("{%v: %v}", h.ID(), h.Addrs()))

	// If this host is a relay node, we need to run libp2p relayv1 service
	var relayService *relayv1.Relay
	if isRelay {
		relayService, err = relayv1.NewRelay(h) // TODO close relayService
		if err != nil {
			return nil, fmt.Errorf("run libp2p relayv1 service error: %w", err)
		}
		klog.Infof("Run as a relay node")
	}

	klog.Infof("Bootstrapping the DHT")
	if err = idht.Bootstrap(ctx); err != nil {
		return nil, fmt.Errorf("failed to bootstrap dht: %w", err)
	}

	// connect to bootstrap
	err = BootstrapConnect(ctx, h, relayPeers)
	if err != nil {
		// We don't want to return error here, so that some
		// edge region that don't have access to the external
		// network can still work
		klog.Warningf("Failed to bootstrap: %v", err)
	}

	// init discovery services
	mdnsPeerChan, err := initMDNS(h, defaultRendezvous)
	if err != nil {
		return nil, fmt.Errorf("init mdns discovery error: %w", err)
	}
	dhtPeerChan, err := initDHT(ctx, idht, defaultRendezvous)
	if err != nil {
		return nil, fmt.Errorf("init dht discovery error: %w", err)
	}

	edgeTunnel := &EdgeTunnel{
		Config:       c,
		p2pHost:      h,
		hostCtx:      ctx,
		nodePeerMap:  make(map[string]peer.ID),
		isRelay:      isRelay,
		relayMaddrs:  relayMaddrs,
		relayPeers:   relayPeers,
		relayService: relayService,
		rendezvous:   defaultRendezvous, // TODO get from config
		mdnsPeerChan: mdnsPeerChan,
		dhtPeerChan:  dhtPeerChan,
		stopCh:       make(chan struct{}),
	}

	h.SetStreamHandler(discoverypb.DiscoveryProtocol, edgeTunnel.discoveryStreamHandler)
	h.SetStreamHandler(proxypb.ProxyProtocol, edgeTunnel.proxyStreamHandler)
	Agent = edgeTunnel // TODO convert var to func
	return edgeTunnel, nil
}

// Register register EdgeTunnel to beehive modules
func Register(c *config.EdgeTunnelConfig, ifm *informers.Manager, mode TunnelMode) error {
	agent, err := newEdgeTunnel(c, ifm, mode)
	if err != nil {
		return fmt.Errorf("register module EdgeTunnel error: %v", err)
	}
	core.Register(agent)
	return nil
}

// Name of EdgeTunnel
func (t *EdgeTunnel) Name() string {
	return modules.EdgeTunnelModuleName
}

// Group of EdgeTunnel
func (t *EdgeTunnel) Group() string {
	return modules.EdgeTunnelModuleName
}

// Enable indicates whether enable this module
func (t *EdgeTunnel) Enable() bool {
	return t.Config.Enable
}

// Start EdgeTunnel
func (t *EdgeTunnel) Start() {
	t.Run()
}
