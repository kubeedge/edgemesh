package tunnel

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
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
	relayv2 "github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/relay"
	"github.com/libp2p/go-msgio/protoio"
	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
	"sigs.k8s.io/yaml"

	"github.com/kubeedge/edgemesh/pkg/apis/config/defaults"
	"github.com/kubeedge/edgemesh/pkg/apis/config/v1alpha1"
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

func (r RelayMap) ContainsPublicIP() bool {
	for _, p := range r {
		for _, addr := range p.Addrs {
			if manet.IsPublicAddr(addr) {
				return true
			}
		}
	}
	return false
}

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

// discovery function is used in the EdgeTunnel to establish connections with other nodes.
// It creates a new stream with the given address information (pi) and discovery type (MDNS or DHT) and performs a handshake.
// If a non-relay node is discovered in DHT discovery, it adds its address to the peerstore to avoid RESERVATION delays.
// Once the connection is established, the function adds the address information of the connection to the node-peer mapping table (t.nodePeerMap) for future communication.
func (t *EdgeTunnel) discovery(discoverType defaults.DiscoveryType, pi peer.AddrInfo) {
	if pi.ID == t.p2pHost.ID() {
		return
	}
	klog.Infof("[%s] Discovery found peer: %s", discoverType, pi)

	// If dht discovery finds a non-relay peer, add the circuit address to this peer.
	// This is done to avoid delays in RESERVATION https://github.com/libp2p/specs/blob/master/relay/circuit-v2.md.
	if discoverType == defaults.DhtDiscovery && !t.isRelayPeer(pi.ID) {
		addrInfo := peer.AddrInfo{ID: pi.ID, Addrs: []ma.Multiaddr{}}
		err := AddCircuitAddrsToPeer(&addrInfo, t.relayMap)
		if err != nil {
			klog.Errorf("Failed to add circuit addrs to peer %s", addrInfo)
			return
		}
		t.p2pHost.Peerstore().AddAddrs(pi.ID, addrInfo.Addrs, peerstore.PermanentAddrTTL)
	}

	if err := t.p2pHost.Connect(t.hostCtx, pi); err != nil {
		klog.Errorf("[%s] Failed to connect to %s, err: %v", discoverType, pi, err)
		return
	}

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

// discoveryStreamHandler handles incoming streams for discovery service.
// It reads the handshake message from the incoming stream and writes a response message,
// then maps the nodeName and peerID of the remote peer to the nodePeerMap of EdgeTunnel.
// This function is called when a new stream is received by the discovery service of EdgeTunnel.
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

// GetProxyStream establishes a new stream with a destination peer, either directly or through a relay node,
// by performing a handshake with the destination peer over the stream to confirm the connection.
// It first looks up the destination peer's ID in a cache, and if not found, generates the peer ID and adds circuit addresses to it.
// It then opens a new stream using the libp2p host, and performs a handshake with the destination peer over the stream.
// If the handshake is successful, it returns a new StreamConn object representing the stream.
// If any errors occur during the process, it returns an error.
func (t *EdgeTunnel) GetProxyStream(opts ProxyOptions) (*StreamConn, error) {
	var destInfo peer.AddrInfo
	var err error

	destName := opts.NodeName
	destID, exists := t.nodePeerMap[destName]
	if !exists {
		destID, err = PeerIDFromString(destName)
		if err != nil {
			return nil, fmt.Errorf("failed to generate peer id for %s err: %w", destName, err)
		}
		destInfo = peer.AddrInfo{ID: destID, Addrs: []ma.Multiaddr{}}
		// mapping nodeName and peerID
		klog.Infof("Could not find peer %s in cache, auto generate peer info: %s", destName, destInfo)
		t.nodePeerMap[destName] = destID
	} else {
		destInfo = t.p2pHost.Peerstore().PeerInfo(destID)
	}
	if err = AddCircuitAddrsToPeer(&destInfo, t.relayMap); err != nil {
		return nil, fmt.Errorf("failed to add circuit addrs to peer %s", destInfo)
	}
	t.p2pHost.Peerstore().AddAddrs(destInfo.ID, destInfo.Addrs, peerstore.PermanentAddrTTL)

	stream, err := t.p2pHost.NewStream(network.WithUseTransient(t.hostCtx, "relay"), destID, defaults.ProxyProtocol)
	if err != nil {
		return nil, fmt.Errorf("new stream between %s: %s err: %w", destName, destInfo, err)
	}
	klog.Infof("New stream between peer %s: %s success", destName, destInfo)
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

// tryDialEndpoint tries to dial to an endpoint with given protocol, ip and port.
// If TCP or UDP protocol is used, it retries several times and waits for DailSleepTime between each try.
// If neither TCP nor UDP is used, it returns an error with an unsupported protocol message.
// when maximum retries are reached for the given protocol, it logs the error and returns it.
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
			time.Sleep(DailSleepTime)
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

// BootstrapConnect tries to connect to a list of bootstrap peers in a relay map.
// The function runs a loop to attempt connecting to each peer, and will retry if some peers fail to connect.
// The function returns an error if it fails to connect to all bootstrap peers after a certain period of time.
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

	for _, bad := range badRelays {
		klog.Warningf("[Bootstrap] bootstrapping to %s : %s timeout", bad, bootstrapPeers[bad])
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
			if relay.ID == t.p2pHost.ID() {
				continue
			}
			select {
			case peerSource <- *relay:
				klog.Infoln("[Finder] send relayMap peer:", relay)
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
		t.connectToRelays("Heartbeat")
		// We make the return value of ConditionFunc, such as bool to return false,
		// and err to return to nil, to ensure that we can continuously execute
		// the ConditionFunc.
		return false, nil
	}, t.stopCh)
	if err != nil {
		klog.Errorf("[Heartbeat] causes an error %v", err)
	}
}

func (t *EdgeTunnel) connectToRelays(connectType string) {
	wg := sync.WaitGroup{}
	for _, relay := range t.relayMap {
		wg.Add(1)
		go func(relay *peer.AddrInfo) {
			defer wg.Done()
			t.connectToRelay(connectType, relay)
		}(relay)
	}
	wg.Wait()
}

func (t *EdgeTunnel) connectToRelay(connectType string, relay *peer.AddrInfo) {
	if t.p2pHost.ID() == relay.ID {
		return
	}
	if len(t.p2pHost.Network().ConnsToPeer(relay.ID)) != 0 {
		klog.Infof("[%s] Already has connection between %s and me", connectType, relay)
		return
	}

	klog.V(0).Infof("[%s] Connection between relay %s is not established, try connect", connectType, relay)
	retryTime := 0
	for retryTime < RetryTime {
		err := t.p2pHost.Connect(t.hostCtx, *relay)
		if err != nil {
			klog.Errorf("[%s] Failed to connect relay %s err: %v", connectType, relay, err)
			time.Sleep(RetryInterval)
			retryTime++
			continue
		}

		klog.Infof("[%s] Success connected to relay %s", connectType, relay)
		break
	}
}

func (t *EdgeTunnel) runConfigWatcher() {
	defer func() {
		if err := t.cfgWatcher.Close(); err != nil {
			klog.Errorf("[Watcher] Failed to close config watcher")
		}
	}()

	for {
		select {
		case event, ok := <-t.cfgWatcher.Events:
			if !ok {
				klog.Errorf("[Watcher] Failed to get events chan")
				continue
			}
			// k8s configmaps uses symlinks, we need this workaround.
			// updating k8s configmaps will delete the file inotify
			if event.Op == fsnotify.Remove {
				// re-add a new watcher pointing to the new symlink/file
				if err := t.cfgWatcher.Add(t.Config.ConfigPath); err != nil {
					klog.Errorf("[Watcher] Failed to re-add watcher in %s, err: %v", t.Config.ConfigPath, err)
					return
				}
				t.doReload(t.Config.ConfigPath)
			}
			// also allow normal files to be modified and reloaded.
			if event.Op&fsnotify.Write == fsnotify.Write {
				t.doReload(t.Config.ConfigPath)
			}
		case err, ok := <-t.cfgWatcher.Errors:
			if !ok {
				klog.Errorf("[Watcher] Failed to get errors chan")
				continue
			}
			klog.Errorf("[Watcher] Config watcher got an error:", err)
		}
	}
}

type reloadConfig struct {
	Modules *struct {
		EdgeTunnelConfig *v1alpha1.EdgeTunnelConfig `json:"edgeTunnel,omitempty"`
	} `json:"modules,omitempty"`
}

func (t *EdgeTunnel) doReload(configPath string) {
	klog.Infof("[Watcher] Reload config from %s", configPath)

	var cfg reloadConfig
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		klog.Errorf("[Watcher] Failed to read config file %s: %v", configPath, err)
		return
	}
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		klog.Errorf("[Watcher] Failed to unmarshal config file %s: %v", configPath, err)
		return
	}

	klog.Infof("[Watcher] Generate new relay map:")
	relayMap := GenerateRelayMap(cfg.Modules.EdgeTunnelConfig.RelayNodes, t.Config.Transport, t.Config.ListenPort)
	for nodeName, pi := range relayMap {
		klog.Infof("%s => %s", nodeName, pi)
	}
	t.relayMap = relayMap

	// enable or disable relayv2 service
	_, exists := t.relayMap[t.Config.NodeName]
	if exists {
		if t.relayService == nil && t.Config.Mode == defaults.ServerClientMode {
			t.relayService, err = relayv2.New(t.p2pHost, relayv2.WithLimit(nil))
			if err != nil {
				klog.Errorf("[Watcher] Failed to enable relayv2 service, err: %v", err)
			} else {
				t.isRelay = true
				klog.Infof("[Watcher] Enable relayv2 service success")
			}
		}
	} else {
		if t.relayService != nil && t.Config.Mode == defaults.ServerClientMode {
			err = t.relayService.Close()
			if err != nil {
				klog.Errorf("[Watcher] Failed to close relayv2 service, err: %v", err)
			} else {
				t.isRelay = false
				t.relayService = nil
				klog.Infof("[Watcher] Disable relayv2 service success")
			}
		}
	}

	t.connectToRelays("Watcher")
}

func (t *EdgeTunnel) Run() {
	go t.runMdnsDiscovery()
	go t.runDhtDiscovery()
	go t.runConfigWatcher()
	t.runHeartbeat()
}
