package controller

import (
	"fmt"
	"sync"

	"github.com/buraksezer/consistent"
	apiv1alpha3 "istio.io/api/networking/v1alpha3"
	istioapi "istio.io/client-go/pkg/apis/networking/v1alpha3"
	istiolisters "istio.io/client-go/pkg/listers/networking/v1alpha3"
	v1 "k8s.io/api/core/v1"
	k8slisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	"github.com/kubeedge/edgemesh/pkg/common/informers"
	"github.com/kubeedge/edgemesh/pkg/gateway/chassis/loadbalancer/consistenthash/hashring"
)

var (
	APIConn *ChassisController
	once    sync.Once
)

type ChassisController struct {
	mu sync.Mutex

	podLister k8slisters.PodLister
	svcLister k8slisters.ServiceLister
	epLister  k8slisters.EndpointsLister
	drLister  istiolisters.DestinationRuleLister

	epInformer cache.SharedIndexInformer
	drInformer cache.SharedIndexInformer
}

func Init(ifm *informers.Manager) {
	once.Do(func() {
		APIConn = &ChassisController{
			podLister:  ifm.GetKubeFactory().Core().V1().Pods().Lister(),
			svcLister:  ifm.GetKubeFactory().Core().V1().Services().Lister(),
			epLister:   ifm.GetKubeFactory().Core().V1().Endpoints().Lister(),
			drLister:   ifm.GetIstioFactory().Networking().V1alpha3().DestinationRules().Lister(),
			epInformer: ifm.GetKubeFactory().Core().V1().Endpoints().Informer(),
			drInformer: ifm.GetIstioFactory().Networking().V1alpha3().DestinationRules().Informer(),
		}
		ifm.RegisterSyncedFunc(APIConn.onCacheSynced)
	})
}

func (c *ChassisController) onCacheSynced() {
	// set informers event handler
	c.epInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{UpdateFunc: c.epUpdate})
	c.drInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.drAdd, UpdateFunc: c.drUpdate, DeleteFunc: c.drDelete})
}

func (c *ChassisController) GetPodLister() k8slisters.PodLister {
	return c.podLister
}

func (c *ChassisController) GetSvcLister() k8slisters.ServiceLister {
	return c.svcLister
}

func (c *ChassisController) GetEpLister() k8slisters.EndpointsLister {
	return c.epLister
}

func (c *ChassisController) GetDrLister() istiolisters.DestinationRuleLister {
	return c.drLister
}

func (c *ChassisController) epUpdate(oldObj, newObj interface{}) {
	eps, ok := newObj.(*v1.Endpoints)
	if !ok {
		klog.Errorf("invalid type %v", newObj)
		return
	}
	// service, endpoints and destinationRule should have the same name
	dr, err := c.drLister.DestinationRules(eps.Namespace).Get(eps.Name)
	if err == nil && dr != nil && isConsistentHashLB(dr) {
		// may need to update hash ring
		c.updateHashRing(eps)
	}
}

func (c *ChassisController) drAdd(obj interface{}) {
	dr, ok := obj.(*istioapi.DestinationRule)
	if !ok {
		klog.Errorf("invalid type %v", obj)
		return
	}
	// If the loadbalance strategy is `consistentHash`, we need to create a hash ring.
	if isConsistentHashLB(dr) {
		c.createHashRing(dr.Namespace, dr.Name)
	}
}

func (c *ChassisController) drUpdate(oldObj, newObj interface{}) {
	oldDr, ok := oldObj.(*istioapi.DestinationRule)
	if !ok {
		klog.Errorf("invalid type %v", oldObj)
		return
	}
	newDr, ok := newObj.(*istioapi.DestinationRule)
	if !ok {
		klog.Errorf("invalid type %v", newObj)
		return
	}

	if isConsistentHashLB(oldDr) && !isConsistentHashLB(newDr) {
		// If the loadbalance strategy is updated, if it is no longer a `consistentHash` strategy,
		// we need to delete the exists hash ring.
		c.deleteHashRing(newDr.Namespace, newDr.Name)
	} else if !isConsistentHashLB(oldDr) && isConsistentHashLB(newDr) {
		// If the loadbalance strategy is updated, and it is a `consistentHash` strategy,
		// we need to create a hash ring.
		c.createHashRing(newDr.Namespace, newDr.Name)
	}
}

func (c *ChassisController) drDelete(obj interface{}) {
	dr, ok := obj.(*istioapi.DestinationRule)
	if !ok {
		klog.Errorf("invalid type %v", obj)
		return
	}
	if isConsistentHashLB(dr) {
		c.deleteHashRing(dr.Namespace, dr.Name)
	}
}

func isConsistentHashLB(dr *istioapi.DestinationRule) (ok bool) {
	switch dr.Spec.TrafficPolicy.LoadBalancer.LbPolicy.(type) {
	case *apiv1alpha3.LoadBalancerSettings_ConsistentHash:
		ok = true
	default:
		ok = false
	}
	return ok
}

// createHashRing create hash ring if needed
func (c *ChassisController) createHashRing(namespace, name string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// check hash ring cache
	key := fmt.Sprintf("%s.%s", namespace, name)
	if _, ok := hashring.GetHashRing(key); ok {
		klog.Warningf("hash ring %s already exists in cache", key)
		return
	}

	// get endpoints
	eps, err := c.epLister.Endpoints(namespace).Get(name)
	if err != nil {
		klog.Errorf("failed to get endpoints %s.%s, reason: %v", namespace, name, err)
		return
	}
	if eps == nil {
		klog.Errorf("endpoints %s.%s is nil", namespace, name)
		return
	}

	// create service instances from endpoints
	var instances []hashring.ServiceInstance
	var instanceName string
	for _, subset := range eps.Subsets {
		for _, addr := range subset.Addresses {
			if addr.TargetRef != nil {
				instanceName = addr.TargetRef.Name
			} else {
				instanceName = addr.IP
			}
			instances = append(instances, hashring.ServiceInstance{
				Namespace:    namespace,
				Name:         name,
				InstanceName: instanceName,
			})
		}
	}

	// create hash ring
	hr := hashring.NewServiceInstanceHashRing(instances)
	// store hash ring
	hashring.AddOrUpdateHashRing(key, hr)
}

// updateHashRing update hash ring by endpoints
func (c *ChassisController) updateHashRing(eps *v1.Endpoints) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if eps == nil {
		klog.Errorf("endpoints is nil")
		return
	}

	// get hash ring
	ring := fmt.Sprintf("%s.%s", eps.Namespace, eps.Name)
	hr, ok := hashring.GetHashRing(ring)
	if !ok {
		klog.Warningf("cannot find hash ring %s, create it", ring)
		c.createHashRing(eps.Namespace, eps.Name)
		return
	}

	// find the difference
	added, deleted := findDiff(eps, hr)

	// update hash ring
	for _, key := range added {
		klog.Infof("add ServiceInstance %s to hash ring %s", key, ring)
		namespace, name, instanceIP, err := hashring.SplitKey(key)
		if err != nil {
			klog.Warningf("failed to split key, reason: %v", err)
			continue
		}
		hr.Add(hashring.ServiceInstance{
			Namespace:    namespace,
			Name:         name,
			InstanceName: instanceIP,
		})
	}
	for _, key := range deleted {
		klog.Infof("delete ServiceInstance %s from hash ring %s", key, ring)
		hr.Remove(key)
	}

	// refresh cache
	if len(added) != 0 || len(deleted) != 0 {
		hashring.AddOrUpdateHashRing(ring, hr)
	}
}

// deleteHashRing delete hash ring if needed
func (c *ChassisController) deleteHashRing(namespace, name string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// check hash ring cache
	key := fmt.Sprintf("%s.%s", namespace, name)
	if _, ok := hashring.GetHashRing(key); !ok {
		klog.Warningf("hash ring %s not exists in cache", key)
		return
	}

	hashring.DeleteHashRing(key)
}

// findDiff look for the difference between v1.Endpoints and HashRing.Members
func findDiff(eps *v1.Endpoints, hr *consistent.Consistent) ([]string, []string) {
	var src, dest []string

	// get source array from hr.Members
	for _, member := range hr.GetMembers() {
		si, ok := member.(hashring.ServiceInstance)
		if !ok {
			klog.Errorf("can't convert to ServiceInstance")
			continue
		}
		src = append(src, si.String())
	}
	klog.V(4).Infof("src: %+v", src)

	// build destination array from endpoints
	var instanceName string
	for _, subset := range eps.Subsets {
		for _, addr := range subset.Addresses {
			if addr.TargetRef != nil {
				instanceName = addr.TargetRef.Name
			} else {
				instanceName = addr.IP
			}
			dest = append(dest, fmt.Sprintf("%s#%s#%s", eps.Namespace, eps.Name, instanceName))
		}
	}
	klog.V(4).Infof("dest: %+v", dest)

	return arrayCompare(src, dest)
}

// arrayCompare finds the difference between two string arrays.
func arrayCompare(src []string, dest []string) ([]string, []string) {
	msrc := make(map[string]byte) // source array set
	mall := make(map[string]byte) // union set
	var set []string              // intersection set

	// 1.Create a map for the source array.
	for _, v := range src {
		msrc[v] = 0
		mall[v] = 0
	}
	// 2.Elements that cannot be stored in the destination array are duplicate elements.
	for _, v := range dest {
		l := len(mall)
		mall[v] = 1
		if l != len(mall) {
			l = len(mall)
		} else {
			set = append(set, v)
		}
	}
	// 3.union - intersection = all variable elements
	for _, v := range set {
		delete(mall, v)
	}
	// 4.Now, mall is a complement set, then we use mall to traverse the source array.
	// The element that can be found is the deleted element, and the element that cannot be found is the added element.
	var added, deleted []string
	for v := range mall {
		_, exist := msrc[v]
		if exist {
			deleted = append(deleted, v)
		} else {
			added = append(added, v)
		}
	}

	return added, deleted
}
