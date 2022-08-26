package tunnel

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kubeedge/beehive/pkg/core"
	"github.com/libp2p/go-libp2p"
	p2phost "github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/routing"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	relayv2 "github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/relay"
	ma "github.com/multiformats/go-multiaddr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	"github.com/kubeedge/edgemesh/agent/pkg/tunnel/config"
	"github.com/kubeedge/edgemesh/agent/pkg/tunnel/proxy"
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
	Config     *config.EdgeTunnelConfig
	ProxySvc   *proxy.ProxyService
	Mode       TunnelMode
	kubeClient kubernetes.Interface

	p2pHost      p2phost.Host       // libp2p host
	hostCtx      context.Context    // ctx governs the lifetime of the libp2p host
	peerMapMutex sync.Mutex         // protect peerMap
	peerMap      map[string]peer.ID // map of Kubernetes node name and peer id

	rendezvous   string // unique string to identify group of nodes
	mdnsPeerChan chan peer.AddrInfo

	relayPeersMutex sync.Mutex // protect relayPeers
	relayPeers      map[string]*peer.AddrInfo
	relayService    *relayv2.Relay

	nodeCacheSynced cache.InformerSynced
	resyncPeriod    time.Duration
	stopCh          chan struct{}
}

func newEdgeTunnel(c *config.EdgeTunnelConfig, ifm *informers.Manager, mode TunnelMode) (*EdgeTunnel, error) {
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
		GenerateTransportOption(c.Transport),
		libp2p.DefaultSecurity,
		libp2p.NATPortMap(),
		libp2p.Routing(func(h p2phost.Host) (routing.PeerRouting, error) {
			idht, err = dht.New(ctx, h)
			return idht, err
		}),
		libp2p.EnableAutoRelay(),
		libp2p.EnableNATService(),
	}

	relayPeers := generateRelayPeer(c.RelayNodes, c.Transport, c.ListenPort)
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

	// If this host is a relay node, we need to run libp2p relayv2 service
	var relayService *relayv2.Relay // TODO close relayService
	if isRelay {
		relayService, err = relayv2.New(h)
		if err != nil {
			return nil, fmt.Errorf("run libp2p relayv2 service error: %w", err)
		}
	}

	edgeTunnel := &EdgeTunnel{
		Config:       c,
		ProxySvc:     proxy.NewProxyService(h),
		Mode:         mode,
		kubeClient:   ifm.GetKubeClient(),
		p2pHost:      h,
		hostCtx:      ctx,
		peerMap:      make(map[string]peer.ID),
		relayPeers:   relayPeers,
		relayService: relayService,
		rendezvous:   defaultRendezvous, // TODO get from config
		mdnsPeerChan: initMDNS(h, defaultRendezvous),
		resyncPeriod: 15 * time.Minute, // TODO get from config
		stopCh:       make(chan struct{}),
	}
	return edgeTunnel, nil
}

func generateRelayPeer(relayNodes []*config.RelayNode, protocol string, listenPort int) map[string]*peer.AddrInfo {
	relayPeers := make(map[string]*peer.AddrInfo)
	for _, relayNode := range relayNodes {
		nodeName := relayNode.NodeName
		peerid, err := PeerIDFromString(nodeName)
		if err != nil {
			klog.Errorf("Failed to generate peer id from %s", nodeName)
			continue
		}
		// TODO It is assumed here that we have checked the validity of the IP.
		addrStrings := make([]string, 0)
		for _, addr := range relayNode.AdvertiseAddress {
			addrStrings = append(addrStrings, GenerateMultiAddrString(protocol, addr, listenPort))
		}
		maddrs, err := StringsToMaddrs(addrStrings)
		if err != nil {
			klog.Errorf("Failed to convert addr strings to maddrs: %v", err)
			continue
		}
		relayPeers[nodeName] = &peer.AddrInfo{
			ID:    peerid,
			Addrs: maddrs,
		}
	}
	return relayPeers
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
