package tunnel

import (
	"context"
	"fmt"
	"time"

	p2phost "github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
	corev1 "k8s.io/api/core/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/informers"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	"github.com/kubeedge/edgemesh/agent/pkg/tunnel/controller"
	mdnsutil "github.com/kubeedge/edgemesh/agent/pkg/tunnel/mdns"
	"github.com/kubeedge/edgemesh/common/constants"
)

const (
	RetryConnectTime     = 3
	RetryConnectDuration = 2 * time.Second
	HeartbeatDuration    = 10 * time.Second
)

func (t *EdgeTunnel) runController(kubeClient kubernetes.Interface, resyncPeriod time.Duration) {
	informerFactory := informers.NewSharedInformerFactory(kubeClient, resyncPeriod)
	t.initNodeController(informerFactory.Core().V1().Nodes(), resyncPeriod)
	go t.runNodeController(t.stopCh)

	informerFactory.Start(t.stopCh)
}

func (t *EdgeTunnel) initNodeController(nodeInformer coreinformers.NodeInformer, resyncPeriod time.Duration) {
	t.nodeSynced = nodeInformer.Informer().HasSynced
	nodeInformer.Informer().AddEventHandlerWithResyncPeriod(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    t.handleAddNode,
			UpdateFunc: t.handleUpdateNode,
			DeleteFunc: t.handleDeleteNode,
		}, resyncPeriod)
}

func (t *EdgeTunnel) runNodeController(stopCh <-chan struct{}) {
	klog.InfoS("Running node controller")

	if !cache.WaitForNamedCacheSync("tunnel node controller", stopCh, t.nodeSynced) {
		klog.Errorf("Failed to wait for cache sync")
		return
	}
}

func (t *EdgeTunnel) addPeer(name string) {
	t.peerMapMutex.Lock()
	defer t.peerMapMutex.Unlock()

	klog.Infof("======== %v", t.peerMap)
	id, err := mdnsutil.PeerIDFromString(name)
	if err != nil {
		klog.ErrorS(err, "Failed to generate peer id for %s", name)
		return
	}
	t.peerMap[name] = &peer.AddrInfo{
		ID:    id,
		Addrs: make([]ma.Multiaddr, 0),
	}
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

func (t *EdgeTunnel) Run() {
	t.runController(t.kubeClient, t.resyncPeriod)

	for {
		relay, err := controller.APIConn.GetPeerAddrInfo(constants.ServerAddrName)
		if err != nil {
			klog.Errorf("Failed to get tunnel server addr: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}

		if len(t.peerHost.Network().ConnsToPeer(relay.ID)) == 0 {
			klog.Warningf("Connection between agent and server %v is not established, try connect", relay.Addrs)
			retryTime := 0
			for retryTime < RetryConnectTime {
				klog.Infof("Tunnel agent connecting to tunnel server")
				err = t.peerHost.Connect(context.Background(), *relay)
				if err != nil {
					klog.Warningf("Connect to server err: %v", err)
					time.Sleep(RetryConnectDuration)
					retryTime++
					continue
				}

				if t.Mode == ServerClientMode {
					err = controller.APIConn.SetPeerAddrInfo(t.Config.NodeName, InfoFrompeerHostAndRelay(t.peerHost, relay))
					if err != nil {
						klog.Warningf("Set peer addr info to secret err: %v", err)
						time.Sleep(RetryConnectDuration)
						retryTime++
						continue
					}
				}

				klog.Infof("agent success connected to server %v", relay.Addrs)
				break
			}
		}
		// heartbeat time
		time.Sleep(HeartbeatDuration)
	}
}

func InfoFrompeerHostAndRelay(host p2phost.Host, relay *peer.AddrInfo) *peer.AddrInfo {
	p2pProto := ma.ProtocolWithCode(ma.P_P2P)
	circuitProto := ma.ProtocolWithCode(ma.P_CIRCUIT)
	peerAddrInfo := &peer.AddrInfo{
		ID:    host.ID(),
		Addrs: host.Addrs(),
	}
	for _, v := range relay.Addrs {
		circuitAddr, err := ma.NewMultiaddr(v.String() + "/" + p2pProto.Name + "/" + relay.ID.String() + "/" + circuitProto.Name)
		if err != nil {
			klog.Warningf("New multi addr err: %v", err)
			continue
		}
		peerAddrInfo.Addrs = append(peerAddrInfo.Addrs, circuitAddr)
	}
	return peerAddrInfo
}
