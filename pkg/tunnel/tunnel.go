package tunnel

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	ds "github.com/ipfs/go-datastore"
	dsync "github.com/ipfs/go-datastore/sync"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-kad-dht/dual"
	p2phost "github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	drouting "github.com/libp2p/go-libp2p/p2p/discovery/routing"
	dutil "github.com/libp2p/go-libp2p/p2p/discovery/util"
	"github.com/libp2p/go-msgio/protoio"
	ma "github.com/multiformats/go-multiaddr"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"

	"github.com/kubeedge/edgemesh/pkg/apis/config/defaults"
	discoverypb "github.com/kubeedge/edgemesh/pkg/tunnel/pb/discovery"
	proxypb "github.com/kubeedge/edgemesh/pkg/tunnel/pb/proxy"
	netutil "github.com/kubeedge/edgemesh/pkg/util/net"
)

const (
	MaxReadSize = 4096

	DailRetryTime = 3
	DailSleepTime = 500 * time.Microsecond

	RetryTime     = 3
	RetryInterval = 2 * time.Second
)

type RelayMap map[string]*peer.AddrInfo

// discoveryNotifee implement mdns interface
type discoveryNotifee struct {
	PeerChan chan peer.AddrInfo
}

// HandlePeerFound interface to be called when new peer is found
func (n *discoveryNotifee) HandlePeerFound(pi peer.AddrInfo) {
	n.PeerChan <- pi
}

// initMDNS initialize the MDNS service
func initMDNS(host p2phost.Host, rendezvous string) (chan peer.AddrInfo, error) {
	n := &discoveryNotifee{}
	n.PeerChan = make(chan peer.AddrInfo)

	ser := mdns.NewMdnsService(host, rendezvous, n)
	if err := ser.Start(); err != nil {
		return nil, err
	}
	klog.Infof("Starting MDNS discovery service")
	return n.PeerChan, nil
}

func (t *EdgeTunnel) runMdnsDiscovery() {
	for pi := range t.mdnsPeerChan {
		t.discovery(defaults.MdnsDiscovery, pi)
	}
}

func initDHT(ctx context.Context, ddht *dual.DHT, rendezvous string) (<-chan peer.AddrInfo, error) {
	routingDiscovery := drouting.NewRoutingDiscovery(ddht)
	dutil.Advertise(ctx, routingDiscovery, rendezvous)
	klog.Infof("Starting DHT discovery service")

	peerChan, err := routingDiscovery.FindPeers(ctx, rendezvous)
	if err != nil {
		return nil, err
	}

	return peerChan, nil
}

func (t *EdgeTunnel) runDhtDiscovery() {
	for pi := range t.dhtPeerChan {
		t.discovery(defaults.DhtDiscovery, pi)
	}
}

func (t *EdgeTunnel) isRelayPeer(id peer.ID) bool {
	for _, relay := range t.relayMap {
		if relay.ID == id {
			return true
		}
	}
	return false
}

func (t *EdgeTunnel) discovery(discoverType defaults.DiscoveryType, pi peer.AddrInfo) {
	if pi.ID == t.p2pHost.ID() {
		return
	}

	// If dht discovery finds a non-relay peer, add the circuit address to this peer.
	// This is done to avoid delays in RESERVATION https://github.com/libp2p/specs/blob/master/relay/circuit-v2.md.
	if discoverType == defaults.DhtDiscovery && !t.isRelayPeer(pi.ID) {
		addrInfo := peer.AddrInfo{ID: pi.ID, Addrs: []ma.Multiaddr{}}
		err := AddCircuitAddrsToPeer(&addrInfo, t.relayMap)
		if err != nil {
			klog.Errorf("Failed to add circuit addrs to peer %s", addrInfo)
			return
		}
		t.p2pHost.Peerstore().AddAddrs(pi.ID, addrInfo.Addrs, peerstore.TempAddrTTL)
	}

	klog.Infof("[%s] Discovery found peer: %s", discoverType, t.p2pHost.Peerstore().PeerInfo(pi.ID))
	stream, err := t.p2pHost.NewStream(network.WithUseTransient(t.hostCtx, "relay"), pi.ID, defaults.DiscoveryProtocol)
	if err != nil {
		klog.Errorf("[%s] New stream between peer %s err: %v", discoverType, pi, err)
		return
	}
	defer func() {
		err = stream.Reset()
		if err != nil {
			klog.Errorf("[%s] Stream between %s reset err: %v", discoverType, pi, err)
		}
	}()
	klog.Infof("[%s] New stream between peer %s success", discoverType, pi)

	streamWriter := protoio.NewDelimitedWriter(stream)
	streamReader := protoio.NewDelimitedReader(stream, MaxReadSize) // TODO get maxSize from default

	// handshake with dest peer
	protocol := string(defaults.MdnsDiscovery)
	if discoverType == defaults.DhtDiscovery {
		protocol = string(defaults.DhtDiscovery)
	}
	msg := &discoverypb.Discovery{
		Type:     discoverypb.Discovery_CONNECT.Enum(),
		Protocol: &protocol,
		NodeName: &t.Config.NodeName,
	}
	err = streamWriter.WriteMsg(msg)
	if err != nil {
		klog.Errorf("[%s] Write msg to %s err: %v", discoverType, pi, err)
		return
	}

	// read response
	msg.Reset()
	err = streamReader.ReadMsg(msg)
	if err != nil {
		klog.Errorf("[%s] Read response msg from %s err: %v", discoverType, pi, err)
		return
	}
	msgType := msg.GetType()
	if msgType != discoverypb.Discovery_SUCCESS {
		klog.Errorf("[%s] Failed to build stream between %s, Type is %s, err: %v", discoverType, pi, msg.GetType(), err)
		return
	}

	// (re)mapping nodeName and peerID
	nodeName := msg.GetNodeName()
	klog.Infof("[%s] Discovery to %s : %s", protocol, nodeName, pi)
	t.nodePeerMap[nodeName] = pi.ID
}

func (t *EdgeTunnel) discoveryStreamHandler(stream network.Stream) {
	remotePeer := peer.AddrInfo{
		ID:    stream.Conn().RemotePeer(),
		Addrs: []ma.Multiaddr{stream.Conn().RemoteMultiaddr()},
	}
	klog.Infof("Discovery service got a new stream from %s", remotePeer)

	streamWriter := protoio.NewDelimitedWriter(stream)
	streamReader := protoio.NewDelimitedReader(stream, MaxReadSize) // TODO get maxSize from default

	// read handshake
	msg := new(discoverypb.Discovery)
	err := streamReader.ReadMsg(msg)
	if err != nil {
		klog.Errorf("Read msg from %s err: %v", remotePeer, err)
		return
	}
	if msg.GetType() != discoverypb.Discovery_CONNECT {
		klog.Errorf("Stream between %s, Type should be CONNECT", remotePeer)
		return
	}

	// write response
	protocol := msg.GetProtocol()
	nodeName := msg.GetNodeName()
	msg.Type = discoverypb.Discovery_SUCCESS.Enum()
	msg.NodeName = &t.Config.NodeName
	err = streamWriter.WriteMsg(msg)
	if err != nil {
		klog.Errorf("[%s] Write msg to %s err: %v", protocol, remotePeer, err)
		return
	}

	// (re)mapping nodeName and peerID
	klog.Infof("[%s] Discovery from %s : %s", protocol, nodeName, remotePeer)
	t.nodePeerMap[nodeName] = remotePeer.ID
}

type ProxyOptions struct {
	Protocol string
	NodeName string
	IP       string
	Port     int32
}

func (t *EdgeTunnel) GetProxyStream(opts ProxyOptions) (*StreamConn, error) {
	destName := opts.NodeName
	destID, exists := t.nodePeerMap[destName]
	if !exists {
		var err error
		destID, err = PeerIDFromString(destName)
		if err != nil {
			return nil, fmt.Errorf("failed to generate peer id for %s err: %w", destName, err)
		}
		destInfo := peer.AddrInfo{ID: destID, Addrs: []ma.Multiaddr{}}
		err = AddCircuitAddrsToPeer(&destInfo, t.relayMap)
		if err != nil {
			return nil, fmt.Errorf("failed to add circuit addrs to peer %s", destInfo)
		}
		t.p2pHost.Peerstore().AddAddrs(destInfo.ID, destInfo.Addrs, peerstore.TempAddrTTL)
		// mapping nodeName and peerID
		klog.Infof("Could not find peer %s in cache, auto generate peer info: %s", destName, t.p2pHost.Peerstore().PeerInfo(destID))
		t.nodePeerMap[destName] = destID
	}

	stream, err := t.p2pHost.NewStream(network.WithUseTransient(t.hostCtx, "relay"), destID, defaults.ProxyProtocol)
	if err != nil {
		return nil, fmt.Errorf("new stream between %s err: %w", destName, err)
	}
	klog.Infof("New stream between peer %s success", t.p2pHost.Peerstore().PeerInfo(destID))
	// defer stream.Close() // will close the stream elsewhere

	streamWriter := protoio.NewDelimitedWriter(stream)
	streamReader := protoio.NewDelimitedReader(stream, MaxReadSize)

	// handshake with dest peer
	msg := &proxypb.Proxy{
		Type:     proxypb.Proxy_CONNECT.Enum(),
		Protocol: &opts.Protocol,
		NodeName: &opts.NodeName,
		Ip:       &opts.IP,
		Port:     &opts.Port,
	}
	if err = streamWriter.WriteMsg(msg); err != nil {
		resetErr := stream.Reset()
		if resetErr != nil {
			return nil, fmt.Errorf("stream between %s reset err: %w", opts.NodeName, resetErr)
		}
		return nil, fmt.Errorf("write conn msg to %s err: %w", opts.NodeName, err)
	}

	// read response
	msg.Reset()
	if err = streamReader.ReadMsg(msg); err != nil {
		resetErr := stream.Reset()
		if resetErr != nil {
			return nil, fmt.Errorf("stream between %s reset err: %w", opts.NodeName, resetErr)
		}
		return nil, fmt.Errorf("read conn result msg from %s err: %w", opts.NodeName, err)
	}
	if msg.GetType() == proxypb.Proxy_FAILED {
		resetErr := stream.Reset()
		if resetErr != nil {
			return nil, fmt.Errorf("stream between %s reset err: %w", opts.NodeName, err)
		}
		return nil, fmt.Errorf("libp2p dial %v err: Proxy.type is %s", opts, msg.GetType())
	}

	msg.Reset()
	klog.V(4).Infof("libp2p dial %v success", opts)

	return NewStreamConn(stream), nil
}

func (t *EdgeTunnel) proxyStreamHandler(stream network.Stream) {
	remotePeer := peer.AddrInfo{
		ID:    stream.Conn().RemotePeer(),
		Addrs: []ma.Multiaddr{stream.Conn().RemoteMultiaddr()},
	}
	klog.Infof("Proxy service got a new stream from %s", remotePeer)

	streamWriter := protoio.NewDelimitedWriter(stream)
	streamReader := protoio.NewDelimitedReader(stream, MaxReadSize) // TODO get maxSize from default

	// read handshake
	msg := new(proxypb.Proxy)
	err := streamReader.ReadMsg(msg)
	if err != nil {
		klog.Errorf("Read msg from %s err: %v", remotePeer, err)
		return
	}
	if msg.GetType() != proxypb.Proxy_CONNECT {
		klog.Errorf("Read msg from %s type should be CONNECT", remotePeer)
		return
	}
	targetProto := msg.GetProtocol()
	targetNode := msg.GetNodeName()
	targetIP := msg.GetIp()
	targetPort := msg.GetPort()
	targetAddr := fmt.Sprintf("%s:%d", targetIP, targetPort)

	proxyConn, err := tryDialEndpoint(targetProto, targetIP, int(targetPort))
	if err != nil {
		klog.Errorf("l4 proxy connect to %v err: %v", msg, err)
		msg.Reset()
		msg.Type = proxypb.Proxy_FAILED.Enum()
		if err = streamWriter.WriteMsg(msg); err != nil {
			klog.Errorf("Write msg to %s err: %v", remotePeer, err)
			return
		}
		return
	}

	// write response
	msg.Type = proxypb.Proxy_SUCCESS.Enum()
	err = streamWriter.WriteMsg(msg)
	if err != nil {
		klog.Errorf("Write msg to %s err: %v", remotePeer, err)
		return
	}
	msg.Reset()

	streamConn := NewStreamConn(stream)
	switch targetProto {
	case TCP:
		go netutil.ProxyConn(streamConn, proxyConn)
	case UDP:
		go netutil.ProxyConnUDP(streamConn, proxyConn.(*net.UDPConn))
	}
	klog.Infof("Success proxy for {%s %s %s}", targetProto, targetNode, targetAddr)
}

func tryDialEndpoint(protocol, ip string, port int) (conn net.Conn, err error) {
	switch protocol {
	case TCP:
		for i := 0; i < DailRetryTime; i++ {
			conn, err = net.DialTCP(TCP, nil, &net.TCPAddr{
				IP:   net.ParseIP(ip),
				Port: port,
			})
			if err == nil {
				return conn, nil
			}
			time.Sleep(DailRetryTime)
		}
	case UDP:
		for i := 0; i < DailRetryTime; i++ {
			conn, err = net.DialUDP(UDP, nil, &net.UDPAddr{
				IP:   net.ParseIP(ip),
				Port: int(port),
			})
			if err == nil {
				return conn, nil
			}
			time.Sleep(DailSleepTime)
		}
	default:
		return nil, fmt.Errorf("unsupported protocol: %s", protocol)
	}
	klog.Errorf("max retries for dial")
	return nil, err
}

func BootstrapConnect(ctx context.Context, ph p2phost.Host, bootstrapPeers RelayMap) error {
	var lock sync.Mutex
	var badRelays []string
	err := wait.PollImmediate(10*time.Second, time.Minute, func() (bool, error) { // TODO get timeout from config
		badRelays = make([]string, 0)
		var wg sync.WaitGroup
		for n, p := range bootstrapPeers {
			if p.ID == ph.ID() {
				continue
			}

			wg.Add(1)
			go func(n string, p *peer.AddrInfo) {
				defer wg.Done()
				klog.Infof("[Bootstrap] bootstrapping to %s", p.ID)

				ph.Peerstore().AddAddrs(p.ID, p.Addrs, peerstore.PermanentAddrTTL)
				if err := ph.Connect(ctx, *p); err != nil {
					klog.Errorf("[Bootstrap] failed to bootstrap with %s: %v", p, err)
					lock.Lock()
					badRelays = append(badRelays, n)
					lock.Unlock()
					return
				}
				klog.Infof("[Bootstrap] success bootstrapped with %s", p)
			}(n, p)
		}
		wg.Wait()
		if len(badRelays) > 0 {
			klog.Errorf("[Bootstrap] Not all bootstrapDail connected, continue bootstrapDail...")
			return false, nil
		}
		return true, nil
	})

	// delete bad relay from relayMap
	for _, bad := range badRelays {
		klog.Warningf("[Bootstrap] bootstrapping to %s : %s timeout, delete it from relayMap", bad, bootstrapPeers[bad])
		delete(bootstrapPeers, bad)
	}
	return err
}

func newDHT(ctx context.Context, host p2phost.Host, relayPeers RelayMap) (*dual.DHT, error) {
	relays := make([]peer.AddrInfo, 0, len(relayPeers))
	for _, relay := range relayPeers {
		relays = append(relays, *relay)
	}
	dstore := dsync.MutexWrap(ds.NewMapDatastore())
	ddht, err := dual.New(
		ctx,
		host,
		dual.DHTOption(
			dht.Concurrency(10),
			dht.Mode(dht.ModeServer),
			dht.Datastore(dstore)),
		dual.WanDHTOption(dht.BootstrapPeers(relays...)),
	)
	if err != nil {
		return nil, err
	}
	return ddht, nil
}

func (t *EdgeTunnel) nodeNameFromPeerID(id peer.ID) (string, bool) {
	for nodeName, peerID := range t.nodePeerMap {
		if peerID == id {
			return nodeName, true
		}
	}
	return "", false
}

func (t *EdgeTunnel) runRelayFinder(ddht *dual.DHT, peerSource chan peer.AddrInfo, period time.Duration) {
	klog.Infof("Starting relay finder")
	err := wait.PollUntil(period, func() (done bool, err error) {
		// ensure peers in same LAN can send [hop]RESERVE to the relay
		for _, relay := range t.relayMap {
			select {
			case peerSource <- *relay:
			case <-t.hostCtx.Done():
				return
			}
		}
		closestPeers, err := ddht.WAN.GetClosestPeers(t.hostCtx, t.p2pHost.ID().String())
		if err != nil {
			if !IsNoFindPeerError(err) {
				klog.Errorf("[Finder] Failed to get closest peers: %v", err)
			}
			return false, nil
		}
		for _, p := range closestPeers {
			addrs := t.p2pHost.Peerstore().Addrs(p)
			if len(addrs) == 0 {
				continue
			}
			dhtPeer := peer.AddrInfo{ID: p, Addrs: addrs}
			klog.Infoln("[Finder] find a relay:", dhtPeer)
			select {
			case peerSource <- dhtPeer:
			case <-t.hostCtx.Done():
				return
			}
			nodeName, exists := t.nodeNameFromPeerID(dhtPeer.ID)
			if exists {
				t.refreshRelayMap(nodeName, &dhtPeer)
			}
		}
		return false, nil
	}, t.stopCh)
	if err != nil {
		klog.Errorf("[Finder] causes an error %v", err)
	}
}

func (t *EdgeTunnel) refreshRelayMap(nodeName string, dhtPeer *peer.AddrInfo) {
	// Will there be a problem when running on a private network?
	// Still need to observe for a while
	dhtPeer.Addrs = FilterPrivateMaddr(dhtPeer.Addrs)
	dhtPeer.Addrs = FilterCircuitMaddr(dhtPeer.Addrs)

	relayInfo, exists := t.relayMap[nodeName]
	if !exists {
		t.relayMap[nodeName] = dhtPeer
		return
	}

	for _, maddr := range dhtPeer.Addrs {
		relayInfo.Addrs = AppendMultiaddrs(relayInfo.Addrs, maddr)
	}
}

func (t *EdgeTunnel) runHeartbeat() {
	err := wait.PollUntil(time.Duration(t.Config.HeartbeatPeriod)*time.Second, func() (done bool, err error) {
		t.connectToRelays()
		// We make the return value of ConditionFunc, such as bool to return false,
		// and err to return to nil, to ensure that we can continuously execute
		// the ConditionFunc.
		return false, nil
	}, t.stopCh)
	if err != nil {
		klog.Errorf("[Heartbeat] causes an error %v", err)
	}
}

func (t *EdgeTunnel) connectToRelays() {
	wg := sync.WaitGroup{}
	for _, relay := range t.relayMap {
		wg.Add(1)
		go func(relay *peer.AddrInfo) {
			defer wg.Done()
			t.connectToRelay(relay)
		}(relay)
	}
	wg.Wait()
}

func (t *EdgeTunnel) connectToRelay(relay *peer.AddrInfo) {
	if t.p2pHost.ID() == relay.ID {
		return
	}
	if len(t.p2pHost.Network().ConnsToPeer(relay.ID)) != 0 {
		return
	}

	klog.V(0).Infof("[Heartbeat] Connection between relay %s is not established, try connect", relay)
	retryTime := 0
	for retryTime < RetryTime {
		err := t.p2pHost.Connect(t.hostCtx, *relay)
		if err != nil {
			klog.Errorf("[Heartbeat] Failed to connect relay %s err: %v", relay, err)
			time.Sleep(RetryInterval)
			retryTime++
			continue
		}

		klog.Infof("[Heartbeat] Success connected to relay %s", relay)
		break
	}
}

func (t *EdgeTunnel) Run() {
	go t.runMdnsDiscovery()
	go t.runDhtDiscovery()
	t.runHeartbeat()
}
