package tunnel

import (
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/informers"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
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
		klog.Errorf("Failed to wait for cache sync")
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

func (t *EdgeTunnel) Run() {
	t.runController(t.kubeClient, t.resyncPeriod)
}
