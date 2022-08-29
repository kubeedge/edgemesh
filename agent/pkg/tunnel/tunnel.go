package tunnel

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	discovery "github.com/libp2p/go-libp2p-discovery"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	"github.com/libp2p/go-libp2p/p2p/host/autorelay"
	"github.com/libp2p/go-msgio/protoio"
	ma "github.com/multiformats/go-multiaddr"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"

	discoverypb "github.com/kubeedge/edgemesh/agent/pkg/tunnel/pb/discovery"
	proxypb "github.com/kubeedge/edgemesh/agent/pkg/tunnel/pb/proxy"
	"github.com/kubeedge/edgemesh/common/libp2p"
	"github.com/kubeedge/edgemesh/common/util"
)

const (
	MaxReadSize  = 4096
	MaxRetryTime = 3
	UDP          = "udp"

	HeartbeatInterval    = time.Minute // TODO get from config
	retryConnectInterval = 2 * time.Second
	retryConnectTime     = 3
)

// discoveryNotifee implement mdns interface
type discoveryNotifee struct {
	PeerChan chan peer.AddrInfo
}

// HandlePeerFound interface to be called when new peer is found
func (n *discoveryNotifee) HandlePeerFound(pi peer.AddrInfo) {
	n.PeerChan <- pi
}

// initMDNS initialize the MDNS service
func initMDNS(peerhost host.Host, rendezvous string) (chan peer.AddrInfo, error) {
	n := &discoveryNotifee{}
	n.PeerChan = make(chan peer.AddrInfo)

	ser := mdns.NewMdnsService(peerhost, rendezvous, n)
	if err := ser.Start(); err != nil {
		return nil, err
	}
	klog.Infof("Starting MDNS discovery service")
	return n.PeerChan, nil
}

func (t *EdgeTunnel) runMdnsDiscovery() {
	for pi := range t.mdnsPeerChan {
		t.discovery(discoverypb.MdnsDiscovery, pi)
	}
}

func initDHT(ctx context.Context, idht *dht.IpfsDHT, rendezvous string) (<-chan peer.AddrInfo, error) {
	routingDiscovery := discovery.NewRoutingDiscovery(idht)
	discovery.Advertise(ctx, routingDiscovery, rendezvous)
	// The default value of autorelay.AdvertiseBootDelay is 15 min, but I hope
	// that the relay nodes can take on the role of RelayV2 as soon as possible.
	autorelay.AdvertiseBootDelay = 15 * time.Second
	autorelay.Advertise(ctx, routingDiscovery)
	klog.Infof("Starting DHT discovery service")

	peerChan, err := routingDiscovery.FindPeers(ctx, rendezvous)
	if err != nil {
		return nil, err
	}

	return peerChan, nil
}

func (t *EdgeTunnel) runDhtDiscovery() {
	for pi := range t.dhtPeerChan {
		t.discovery(discoverypb.DhtDiscovery, pi)
	}
}

func (t *EdgeTunnel) discovery(discoverType discoverypb.DiscoveryType, pi peer.AddrInfo) {
	if pi.ID == t.p2pHost.ID() {
		return
	}

	klog.Infof("[%s] Discovery found peer: %s", discoverType, pi)
	stream, err := t.p2pHost.NewStream(network.WithUseTransient(t.hostCtx, "for-relay"), pi.ID, discoverypb.DiscoveryProtocol)
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
	protocol := string(discoverypb.MdnsDiscovery)
	if discoverType == discoverypb.DhtDiscovery {
		protocol = string(discoverypb.DhtDiscovery)
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
	t.mu.Lock()
	t.nodePeerMap[nodeName] = pi.ID
	t.mu.Unlock()
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
	t.mu.Lock()
	t.nodePeerMap[nodeName] = remotePeer.ID
	t.mu.Unlock()
}

func (t *EdgeTunnel) GetProxyStream(opts proxypb.ProxyOptions) (*libp2p.StreamConn, error) {
	destName := opts.NodeName
	destID, exists := t.nodePeerMap[destName]
	if !exists {
		klog.Warningf("Could not find %s peer in cache", destName)
		var err error
		destID, err = PeerIDFromString(destName)
		if err != nil {
			return nil, fmt.Errorf("failed to generate peer id for %s err: %w", destName, err)
		}
		destInfo := &peer.AddrInfo{destID, []ma.Multiaddr{}}
		err = AddCircuitAddrsToPeer(destInfo, t.relayPeers)
		if err != nil {
			return nil, fmt.Errorf("failed to add circuit addrs to peer %s", destInfo)
		}
		t.p2pHost.Peerstore().AddAddrs(destInfo.ID, destInfo.Addrs, peerstore.PermanentAddrTTL)
		klog.Infof("Auto generate peer info: %s", t.p2pHost.Peerstore().PeerInfo(destID))
		// mapping nodeName and peerID
		t.nodePeerMap[destName] = destID
	}

	stream, err := t.p2pHost.NewStream(network.WithUseTransient(t.hostCtx, "for-relay"), destID, proxypb.ProxyProtocol)
	if err != nil {
		return nil, fmt.Errorf("new stream between %s err: %w", destName, err)
	}
	klog.Infof("New stream between peer %s success", destID)
	// Will close the stream elsewhere
	// defer stream.Close()

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

	return libp2p.NewStreamConn(stream), nil
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

	streamConn := libp2p.NewStreamConn(stream)
	switch targetProto {
	case TCP:
		go util.ProxyConn(streamConn, proxyConn)
	case UDP:
		go util.ProxyConnUDP(streamConn, proxyConn.(*net.UDPConn))
	}
	klog.Infof("Success proxy for {%s %s %s}", targetProto, targetNode, targetAddr)
}

func tryDialEndpoint(protocol, ip string, port int) (conn net.Conn, err error) {
	switch protocol {
	case TCP:
		for i := 0; i < MaxRetryTime; i++ {
			conn, err = net.DialTCP(TCP, nil, &net.TCPAddr{
				IP:   net.ParseIP(ip),
				Port: port,
			})
			if err == nil {
				return conn, nil
			}
			time.Sleep(time.Second)
		}
	case UDP:
		for i := 0; i < MaxRetryTime; i++ {
			conn, err = net.DialUDP(UDP, nil, &net.UDPAddr{
				IP:   net.ParseIP(ip),
				Port: int(port),
			})
			if err == nil {
				return conn, nil
			}
			time.Sleep(time.Second)
		}
	default:
		return nil, fmt.Errorf("unsupported protocol: %s", protocol)
	}
	klog.Errorf("max retries for dial")
	return nil, err
}

// TODO replace with async.Runner
func (t *EdgeTunnel) Heartbeat() {
	err := wait.PollUntil(HeartbeatInterval, func() (done bool, err error) {
		t.connectToRelays()
		// We make the return value of ConditionFunc, such as bool to return false, and err to return to nil,
		// to ensure that we can continuously execute the ConditionFunc.
		return false, nil
	}, t.stopCh)
	if err != nil {
		klog.Errorf("Heartbeat causes an unknown error %v", err)
	}
}

func (t *EdgeTunnel) connectToRelays() {
	wg := sync.WaitGroup{}
	for _, relay := range t.relayPeers {
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

	klog.V(0).Infof("Connection between relay %s is not established, try connect", relay)
	retryTime := 0
	for retryTime < retryConnectTime {
		err := t.p2pHost.Connect(t.hostCtx, *relay)
		if err != nil {
			klog.Errorf("Failed to connect relay %s err: %v", relay, err)
			time.Sleep(retryConnectInterval)
			retryTime++
			continue
		}

		klog.Infof("Success connected to relay %s", relay)
		break
	}
}

func (t *EdgeTunnel) Run() {
	go t.runMdnsDiscovery()
	go t.runDhtDiscovery()
	t.Heartbeat()
}
