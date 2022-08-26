package tunnel

import (
	"fmt"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	"github.com/libp2p/go-msgio/protoio"
	ma "github.com/multiformats/go-multiaddr"
	corev1 "k8s.io/api/core/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	discoverypb "github.com/kubeedge/edgemesh/agent/pkg/tunnel/pb/discovery"
)

const (
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
func initMDNS(peerhost host.Host, rendezvous string) chan peer.AddrInfo {
	n := &discoveryNotifee{}
	n.PeerChan = make(chan peer.AddrInfo)

	ser := mdns.NewMdnsService(peerhost, rendezvous, n)
	if err := ser.Start(); err != nil {
		panic(err) // TODO return error
	}
	return n.PeerChan
}

func (t *EdgeTunnel) runMdnsDiscovery() {
	for pi := range t.mdnsPeerChan {
		if pi.ID == t.p2pHost.ID() {
			continue
		}
		klog.Infof("[MDNS] Discovery found peer: %s", pi)
		err := t.p2pHost.Connect(t.hostCtx, pi)
		if err != nil {
			klog.Errorf("[MDNS] Connection with peer %s failed: %v", pi, err)
			return
		}
		klog.Infof("[MDNS] Connection with peer %s success", pi)
		stream, err := t.p2pHost.NewStream(t.hostCtx, pi.ID, discoverypb.DiscoveryProtocol)
		if err != nil {
			klog.Errorf("[MDNS] New stream between peer %s err: %v", pi, err)
			return
		}
		// defer stream.Reset() here will cause resource leak

		streamWriter := protoio.NewDelimitedWriter(stream)
		streamReader := protoio.NewDelimitedReader(stream, 4096) // todo get from default

		// handshake with target peer
		protocol := string(discoverypb.MdnsDiscovery)
		msg := &discoverypb.Discovery{
			Type:     discoverypb.Discovery_CONNECT.Enum(),
			Protocol: &protocol,
			NodeName: &t.Config.NodeName,
		}
		err = streamWriter.WriteMsg(msg)
		if err != nil {
			resetErr := stream.Reset()
			if resetErr != nil {
				klog.Errorf("[MDNS] Stream between %s reset err: %v", pi, resetErr)
			}
			klog.Errorf("[MDNS] Write msg to %s err: %v", pi, err)
			return
		}

		// read response
		msg.Reset()
		err = streamReader.ReadMsg(msg)
		if err != nil {
			resetErr := stream.Reset()
			if resetErr != nil {
				klog.Errorf("[MDNS] Stream between %s reset err: %v", pi, resetErr)
			}
			klog.Errorf("[MDNS] Read response msg from %s err: %v", pi, err)
			return
		}
		msgType := msg.GetType()
		if msgType != discoverypb.Discovery_SUCCESS {
			resetErr := stream.Reset()
			if resetErr != nil {
				klog.Errorf("[MDNS] Stream between %s reset err: %v", pi, resetErr)
			}
			klog.Errorf("[MDNS] Failed to build stream between %s, Type is %s, err: %v", pi, msg.GetType(), err)
			return
		}

		// store info
		nodeName := msg.GetNodeName()
		klog.Infof("[%s] discovery to %s : %s", protocol, nodeName, pi)
		// TODO store nodePeer and peerNode info

		// reset resource
		msg.Reset()
		err = stream.Reset()
		if err != nil {
			klog.Errorf("[MDNS] Stream between %s reset err: %v", pi, err)
		}
	}
}

func (t *EdgeTunnel) discoveryStreamHandler(stream network.Stream) {
	remotePeer := peer.AddrInfo{
		ID:    stream.Conn().RemotePeer(),
		Addrs: []ma.Multiaddr{stream.Conn().RemoteMultiaddr()},
	}
	klog.Infof("Discovery service got a new stream from %s", remotePeer)
	streamWriter := protoio.NewDelimitedWriter(stream)
	streamReader := protoio.NewDelimitedReader(stream, 4096) // TODO get from default

	msg := new(discoverypb.Discovery)
	defer msg.Reset()

	err := streamReader.ReadMsg(msg)
	if err != nil {
		klog.Errorf("Read msg from %s err: %v", remotePeer, err)
		return
	}
	if msg.GetType() != discoverypb.Discovery_CONNECT {
		klog.Errorf("Stream between %s, Type should be CONNECT", remotePeer)
		return
	}
	protocol := msg.GetProtocol()
	nodeName := msg.GetNodeName()
	klog.Infof("[%s] discovery from %s : %s", protocol, nodeName, remotePeer)

	// write response
	msg.Type = discoverypb.Discovery_SUCCESS.Enum()
	msg.NodeName = &t.Config.NodeName
	err = streamWriter.WriteMsg(msg)
	if err != nil {
		klog.Errorf("[%s] Write msg to %s err: %v", protocol, remotePeer, err)
		return
	}

	// TODO store nodePeer and peerNode info
}

func (t *EdgeTunnel) runController() {
	informerFactory := informers.NewSharedInformerFactory(t.kubeClient, t.resyncPeriod)
	t.initNodeController(informerFactory.Core().V1().Nodes(), t.resyncPeriod)
	go t.runNodeController(t.stopCh)

	informerFactory.Start(t.stopCh)
}

func (t *EdgeTunnel) initNodeController(nodeInformer coreinformers.NodeInformer, resyncPeriod time.Duration) {
	t.nodeCacheSynced = nodeInformer.Informer().HasSynced
	nodeInformer.Informer().AddEventHandlerWithResyncPeriod(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    t.handleAddNode,
			UpdateFunc: t.handleUpdateNode,
			DeleteFunc: t.handleDeleteNode,
		}, resyncPeriod)
}

func (t *EdgeTunnel) runNodeController(stopCh <-chan struct{}) {
	if !cache.WaitForNamedCacheSync("tunnel node controller", stopCh, t.nodeCacheSynced) {
		return
	}
}

func (t *EdgeTunnel) addPeer(name string) {
	t.peerMapMutex.Lock()
	defer t.peerMapMutex.Unlock()

	id, err := PeerIDFromString(name)
	if err != nil {
		klog.ErrorS(err, "Failed to generate peer id for %s", name)
		return
	}
	t.peerMap[name] = id
}

func (t *EdgeTunnel) delPeer(name string) {
	t.peerMapMutex.Lock()
	delete(t.peerMap, name)
	t.peerMapMutex.Unlock()
}

func (t *EdgeTunnel) handleAddNode(obj interface{}) {
	node, ok := obj.(*corev1.Node)
	if !ok {
		utilruntime.HandleError(fmt.Errorf("unexpected object type: %v", obj))
		return
	}
	klog.Infof("======== %v", node.Name)

	_, exists := t.peerMap[node.Name]
	if !exists {
		t.addPeer(node.Name)
	}
}

func (t *EdgeTunnel) handleUpdateNode(oldObj, newObj interface{}) {
	oldNode, ok := oldObj.(*corev1.Node)
	if !ok {
		utilruntime.HandleError(fmt.Errorf("unexpected object type: %v", oldNode))
		return
	}
	newNode, ok := newObj.(*corev1.Node)
	if !ok {
		utilruntime.HandleError(fmt.Errorf("unexpected object type: %v", newNode))
		return
	}

	if oldNode.Name != newNode.Name {
		t.delPeer(oldNode.Name)
	}
	_, exists := t.peerMap[newNode.Name]
	if !exists {
		t.addPeer(newNode.Name)
	}
}

func (t *EdgeTunnel) handleDeleteNode(obj interface{}) {
	node, ok := obj.(*corev1.Node)
	if !ok {
		utilruntime.HandleError(fmt.Errorf("unexpected object type: %v", obj))
		return
	}
	t.delPeer(node.Name)
}

// TODO replace with async.Runner
func (t *EdgeTunnel) Heartbeat() {
	err := wait.PollImmediateUntil(HeartbeatInterval, func() (done bool, err error) {
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
	t.runController()
	go t.runMdnsDiscovery()
	t.Heartbeat()
}
