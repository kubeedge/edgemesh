package loadbalancer

import (
	"fmt"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	istioapi "istio.io/client-go/pkg/apis/networking/v1alpha3"
	istio "istio.io/client-go/pkg/clientset/versioned"
	istioinformers "istio.io/client-go/pkg/informers/externalversions"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/proxy"
	"k8s.io/kubernetes/pkg/proxy/userspace"
	utilproxy "k8s.io/kubernetes/pkg/proxy/util"
	netutils "k8s.io/utils/net"
	stringslices "k8s.io/utils/strings/slices"

	"github.com/kubeedge/edgemesh/pkg/apis/config/defaults"
	"github.com/kubeedge/edgemesh/pkg/apis/config/v1alpha1"
	"github.com/kubeedge/edgemesh/pkg/tunnel"
	netutil "github.com/kubeedge/edgemesh/pkg/util/net"
)

type ServiceObject struct {
	ip                  net.IP
	port                int
	protocol            v1.Protocol
	nodePort            int
	loadBalancerStatus  v1.LoadBalancerStatus
	serviceAffinityType v1.ServiceAffinity
	stickyMaxAgeSeconds int
}

type affinityState struct {
	clientIP string
	endpoint string
	lastUsed time.Time
}

type affinityPolicy struct {
	affinityType v1.ServiceAffinity
	affinityMap  map[string]*affinityState // map client IP -> affinity info
	ttlSeconds   int
}

func newAffinityPolicy(affinityType v1.ServiceAffinity, ttlSeconds int) *affinityPolicy {
	return &affinityPolicy{
		affinityType: affinityType,
		affinityMap:  make(map[string]*affinityState),
		ttlSeconds:   ttlSeconds,
	}
}

type balancerState struct {
	endpoints []string // a list of "nodeName:podName:ip:port" style strings, nodeName and podName can be empty!!!
	index     int      // current index into endpoints
	affinity  affinityPolicy
}

// assert LoadBalancer is an userspace.LoadBalancer
var _ userspace.LoadBalancer = &LoadBalancer{}

type LoadBalancer struct {
	Config      *v1alpha1.LoadBalancer
	kubeClient  kubernetes.Interface
	istioClient istio.Interface
	syncPeriod  time.Duration
	mu          sync.Mutex // protects serviceMap
	serviceMap  map[proxy.ServicePortName]*ServiceObject
	lock        sync.RWMutex // protects services
	services    map[proxy.ServicePortName]*balancerState
	policyMutex sync.Mutex // protects policyMap
	policyMap   map[proxy.ServicePortName]Policy
	stopCh      chan struct{}
}

func New(config *v1alpha1.LoadBalancer, kubeClient kubernetes.Interface, istioClient istio.Interface, syncPeriod time.Duration) *LoadBalancer {
	return &LoadBalancer{
		Config:      config,
		kubeClient:  kubeClient,
		istioClient: istioClient,
		syncPeriod:  syncPeriod,
		serviceMap:  make(map[proxy.ServicePortName]*ServiceObject),
		services:    make(map[proxy.ServicePortName]*balancerState),
		policyMap:   make(map[proxy.ServicePortName]Policy),
		stopCh:      make(chan struct{}),
	}
}

func (lb *LoadBalancer) Run() error {
	if lb.Config.Caller == defaults.GatewayCaller {
		kubeInformerFactory := informers.NewSharedInformerFactory(lb.kubeClient, lb.syncPeriod)
		serviceInformer := kubeInformerFactory.Core().V1().Services()
		serviceInformer.Informer().AddEventHandlerWithResyncPeriod(
			cache.ResourceEventHandlerFuncs{
				AddFunc:    lb.handleAddService,
				UpdateFunc: lb.handleUpdateService,
				DeleteFunc: lb.handleDeleteService,
			},
			lb.syncPeriod,
		)
		go lb.runService(serviceInformer.Informer().HasSynced, lb.stopCh)
		endpointsInformer := kubeInformerFactory.Core().V1().Endpoints()
		endpointsInformer.Informer().AddEventHandlerWithResyncPeriod(
			cache.ResourceEventHandlerFuncs{
				AddFunc:    lb.handleAddEndpoints,
				UpdateFunc: lb.handleUpdateEndpoints,
				DeleteFunc: lb.handleDeleteEndpoints,
			},
			lb.syncPeriod,
		)
		go lb.runEndpoints(endpointsInformer.Informer().HasSynced, lb.stopCh)
		kubeInformerFactory.Start(lb.stopCh)
	}

	istioInformerFactory := istioinformers.NewSharedInformerFactory(lb.istioClient, lb.syncPeriod)
	drInformer := istioInformerFactory.Networking().V1alpha3().DestinationRules()
	drInformer.Informer().AddEventHandlerWithResyncPeriod(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    lb.handleAddDestinationRule,
			UpdateFunc: lb.handleUpdateDestinationRule,
			DeleteFunc: lb.handleDeleteDestinationRule,
		},
		lb.syncPeriod,
	)
	go lb.runDestinationRule(drInformer.Informer().HasSynced, lb.stopCh)
	istioInformerFactory.Start(lb.stopCh)

	return nil
}

func (lb *LoadBalancer) runService(listerSynced cache.InformerSynced, stopCh <-chan struct{}) {
	klog.InfoS("Starting loadBalancer service controller")

	if !cache.WaitForNamedCacheSync("loadBalancer service", stopCh, listerSynced) {
		return
	}
	lb.OnServiceSynced()
}

func (lb *LoadBalancer) handleAddService(obj interface{}) {
	service, ok := obj.(*v1.Service)
	if !ok {
		utilruntime.HandleError(fmt.Errorf("unexpected object type: %v", obj))
		return
	}
	lb.OnServiceAdd(service)
}

func (lb *LoadBalancer) handleUpdateService(oldObj, newObj interface{}) {
	oldService, ok := oldObj.(*v1.Service)
	if !ok {
		utilruntime.HandleError(fmt.Errorf("unexpected object type: %v", oldObj))
		return
	}
	service, ok := newObj.(*v1.Service)
	if !ok {
		utilruntime.HandleError(fmt.Errorf("unexpected object type: %v", newObj))
		return
	}
	lb.OnServiceUpdate(oldService, service)
}

func (lb *LoadBalancer) handleDeleteService(obj interface{}) {
	service, ok := obj.(*v1.Service)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("unexpected object type: %v", obj))
			return
		}
		if service, ok = tombstone.Obj.(*v1.Service); !ok {
			utilruntime.HandleError(fmt.Errorf("unexpected object type: %v", obj))
			return
		}
	}
	lb.OnServiceDelete(service)
}

func (lb *LoadBalancer) runEndpoints(listerSynced cache.InformerSynced, stopCh <-chan struct{}) {
	klog.InfoS("Starting loadBalancer endpoints controller")

	if !cache.WaitForNamedCacheSync("loadBalancer endpoints", stopCh, listerSynced) {
		return
	}
	lb.OnEndpointsSynced()
}

func (lb *LoadBalancer) handleAddEndpoints(obj interface{}) {
	endpoints, ok := obj.(*v1.Endpoints)
	if !ok {
		utilruntime.HandleError(fmt.Errorf("unexpected object type: %v", obj))
		return
	}
	lb.OnEndpointsAdd(endpoints)
}

func (lb *LoadBalancer) handleUpdateEndpoints(oldObj, newObj interface{}) {
	oldEndpoints, ok := oldObj.(*v1.Endpoints)
	if !ok {
		utilruntime.HandleError(fmt.Errorf("unexpected object type: %v", oldObj))
		return
	}
	endpoints, ok := newObj.(*v1.Endpoints)
	if !ok {
		utilruntime.HandleError(fmt.Errorf("unexpected object type: %v", newObj))
		return
	}
	lb.OnEndpointsUpdate(oldEndpoints, endpoints)
}

func (lb *LoadBalancer) handleDeleteEndpoints(obj interface{}) {
	endpoints, ok := obj.(*v1.Endpoints)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("unexpected object type: %v", obj))
			return
		}
		if endpoints, ok = tombstone.Obj.(*v1.Endpoints); !ok {
			utilruntime.HandleError(fmt.Errorf("unexpected object type: %v", obj))
			return
		}
	}
	lb.OnEndpointsDelete(endpoints)
}

func (lb *LoadBalancer) runDestinationRule(listerSynced cache.InformerSynced, stopCh <-chan struct{}) {
	klog.InfoS("Starting loadBalancer destinationRule controller")

	if !cache.WaitForNamedCacheSync("loadBalancer destinationRule", stopCh, listerSynced) {
		return
	}
	lb.OnDestinationRuleSynced()
}

func (lb *LoadBalancer) handleAddDestinationRule(obj interface{}) {
	dr, ok := obj.(*istioapi.DestinationRule)
	if !ok {
		utilruntime.HandleError(fmt.Errorf("unexpected object type: %v", obj))
		return
	}
	lb.OnDestinationRuleAdd(dr)
}

func (lb *LoadBalancer) handleUpdateDestinationRule(oldObj, newObj interface{}) {
	oldDr, ok := oldObj.(*istioapi.DestinationRule)
	if !ok {
		utilruntime.HandleError(fmt.Errorf("unexpected object type: %v", oldObj))
		return
	}
	dr, ok := newObj.(*istioapi.DestinationRule)
	if !ok {
		utilruntime.HandleError(fmt.Errorf("unexpected object type: %v", newObj))
		return
	}
	lb.OnDestinationRuleUpdate(oldDr, dr)
}

func (lb *LoadBalancer) handleDeleteDestinationRule(obj interface{}) {
	dr, ok := obj.(*istioapi.DestinationRule)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("unexpected object type: %v", obj))
			return
		}
		if dr, ok = tombstone.Obj.(*istioapi.DestinationRule); !ok {
			utilruntime.HandleError(fmt.Errorf("unexpected object type: %v", obj))
			return
		}
	}
	lb.OnDestinationRuleDelete(dr)
}

func (lb *LoadBalancer) newServiceObject(service *v1.Service) {
	if service == nil {
		return
	}
	if utilproxy.ShouldSkipService(service) {
		return
	}
	svcName := types.NamespacedName{Namespace: service.Namespace, Name: service.Name}
	for i := range service.Spec.Ports {
		servicePort := &service.Spec.Ports[i]
		serviceName := proxy.ServicePortName{NamespacedName: svcName, Port: servicePort.Name}
		serviceIP := netutils.ParseIPSloppy(service.Spec.ClusterIP)
		// Kube-apiserver side guarantees SessionAffinityConfig won't be nil when session affinity type is ClientIP
		var stickyMaxAgeSeconds int
		if service.Spec.SessionAffinity == v1.ServiceAffinityClientIP {
			stickyMaxAgeSeconds = int(*service.Spec.SessionAffinityConfig.ClientIP.TimeoutSeconds)
		}
		so := &ServiceObject{
			ip:                  serviceIP,
			port:                int(servicePort.Port),
			protocol:            servicePort.Protocol,
			nodePort:            int(servicePort.NodePort),
			loadBalancerStatus:  *service.Status.LoadBalancer.DeepCopy(),
			serviceAffinityType: v1.ServiceAffinityNone,
			stickyMaxAgeSeconds: stickyMaxAgeSeconds,
		}
		lb.serviceMap[serviceName] = so

		err := lb.NewService(serviceName, so.serviceAffinityType, so.stickyMaxAgeSeconds)
		if err != nil {
			klog.ErrorS(err, "Failed to new service", "serviceName", serviceName)
		}
	}
}

func (lb *LoadBalancer) OnServiceAdd(service *v1.Service) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	lb.newServiceObject(service)
	lb.cleanupStaleStickySessions()
}

func (lb *LoadBalancer) OnServiceUpdate(oldService, service *v1.Service) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	lb.newServiceObject(service)
	lb.cleanupStaleStickySessions()
}

func (lb *LoadBalancer) OnServiceDelete(service *v1.Service) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	if service == nil {
		return
	}
	if utilproxy.ShouldSkipService(service) {
		return
	}
	svcName := types.NamespacedName{Namespace: service.Namespace, Name: service.Name}
	for i := range service.Spec.Ports {
		servicePort := &service.Spec.Ports[i]
		serviceName := proxy.ServicePortName{NamespacedName: svcName, Port: servicePort.Name}
		delete(lb.serviceMap, serviceName)
		lb.DeleteService(serviceName)
	}
	lb.cleanupStaleStickySessions()
}

func (lb *LoadBalancer) OnServiceSynced() {

}

// clean up any stale sticky session records in the hash map.
func (lb *LoadBalancer) cleanupStaleStickySessions() {
	for name := range lb.serviceMap {
		lb.CleanupStaleStickySessions(name)
	}
}

// Loop through the valid endpoints and then the endpoints associated with the Load Balancer.
// Then remove any session affinity records that are not in both lists.
// This assumes the lb.lock is held.
func (lb *LoadBalancer) removeStaleAffinity(svcPort proxy.ServicePortName, newEndpoints []string) {
	newEndpointsSet := sets.NewString()
	for _, newEndpoint := range newEndpoints {
		newEndpointsSet.Insert(newEndpoint)
	}

	state, exists := lb.services[svcPort]
	if !exists {
		return
	}
	for _, existingEndpoint := range state.endpoints {
		if !newEndpointsSet.Has(existingEndpoint) {
			klog.V(2).InfoS("Delete endpoint for service", "endpoint", existingEndpoint, "servicePortName", svcPort)
			removeSessionAffinityByEndpoint(state, svcPort, existingEndpoint)
		}
	}
}

func (lb *LoadBalancer) OnEndpointsAdd(endpoints *v1.Endpoints) {
	portsToEndpoints := buildPortsToEndpointsMap(endpoints)

	lb.lock.Lock()
	defer lb.lock.Unlock()

	svcName := types.NamespacedName{Namespace: endpoints.Namespace, Name: endpoints.Name}
	for portname := range portsToEndpoints {
		svcPort := proxy.ServicePortName{NamespacedName: svcName, Port: portname}
		newEndpoints := portsToEndpoints[portname]
		state, exists := lb.services[svcPort]

		if !exists || state == nil || len(newEndpoints) > 0 {
			klog.V(1).InfoS("Setting endpoints service", "servicePortName", svcPort, "endpoints", newEndpoints)
			// OnEndpointsAdd can be called without NewService being called externally.
			// To be safe we will call it here. A new service will only be created
			// if one does not already exist.
			state = lb.newServiceInternal(svcPort, v1.ServiceAffinity(""), 0)
			state.endpoints = utilproxy.ShuffleStrings(newEndpoints)

			// Reset the round-robin index.
			state.index = 0

			// Sync the backend loadBalancer policy.
			lb.policyMutex.Lock()
			if policy, exists := lb.policyMap[svcPort]; exists {
				policy.Sync(newEndpoints)
			}
			lb.policyMutex.Unlock()
		}
	}
}

func (lb *LoadBalancer) OnEndpointsUpdate(oldEndpoints *v1.Endpoints, endpoints *v1.Endpoints) {
	portsToEndpoints := buildPortsToEndpointsMap(endpoints)
	oldPortsToEndpoints := buildPortsToEndpointsMap(oldEndpoints)
	registeredEndpoints := make(map[proxy.ServicePortName]bool)

	lb.lock.Lock()
	defer lb.lock.Unlock()

	svcName := types.NamespacedName{Namespace: endpoints.Namespace, Name: endpoints.Name}
	for portname := range portsToEndpoints {
		svcPort := proxy.ServicePortName{NamespacedName: svcName, Port: portname}
		newEndpoints := portsToEndpoints[portname]
		state, exists := lb.services[svcPort]

		curEndpoints := []string{}
		if state != nil {
			curEndpoints = state.endpoints
		}

		if !exists || state == nil || len(curEndpoints) != len(newEndpoints) || !slicesEquiv(stringslices.Clone(curEndpoints), newEndpoints) {
			klog.V(1).InfoS("LoadBalancerEX: Setting endpoints for service", "servicePortName", svcPort, "endpoints", newEndpoints)
			lb.removeStaleAffinity(svcPort, newEndpoints)
			// OnEndpointsUpdate can be called without NewService being called externally.
			// To be safe we will call it here.  A new service will only be created
			// if one does not already exist.  The affinity will be updated
			// later, once NewService is called.
			state = lb.newServiceInternal(svcPort, v1.ServiceAffinity(""), 0)
			state.endpoints = utilproxy.ShuffleStrings(newEndpoints)

			// Reset the round-robin index.
			state.index = 0

			// Sync the backend loadBalancer policy.
			lb.policyMutex.Lock()
			if policy, exists := lb.policyMap[svcPort]; exists {
				policy.Sync(newEndpoints)
			}
			lb.policyMutex.Unlock()
		}
		registeredEndpoints[svcPort] = true
	}

	// Now remove all endpoints missing from the update.
	for portname := range oldPortsToEndpoints {
		svcPort := proxy.ServicePortName{NamespacedName: svcName, Port: portname}
		if _, exists := registeredEndpoints[svcPort]; !exists {
			lb.resetService(svcPort)

			// Sync the backend loadBalancer policy.
			lb.policyMutex.Lock()
			if policy, exists := lb.policyMap[svcPort]; exists {
				policy.Sync(nil)
			}
			lb.policyMutex.Unlock()
		}
	}
}

func (lb *LoadBalancer) resetService(svcPort proxy.ServicePortName) {
	// If the service is still around, reset but don't delete.
	if state, ok := lb.services[svcPort]; ok {
		if len(state.endpoints) > 0 {
			klog.V(2).InfoS("Removing endpoints service", "servicePortName", svcPort)
			state.endpoints = []string{}
		}
		state.index = 0
		state.affinity.affinityMap = map[string]*affinityState{}
	}
}

func (lb *LoadBalancer) OnEndpointsDelete(endpoints *v1.Endpoints) {
	portsToEndpoints := buildPortsToEndpointsMap(endpoints)

	lb.lock.Lock()
	defer lb.lock.Unlock()

	for portname := range portsToEndpoints {
		svcPort := proxy.ServicePortName{NamespacedName: types.NamespacedName{Namespace: endpoints.Namespace, Name: endpoints.Name}, Port: portname}
		lb.resetService(svcPort)

		// Sync the backend loadBalancer policy.
		lb.policyMutex.Lock()
		if policy, exists := lb.policyMap[svcPort]; exists {
			policy.Sync(nil)
		}
		lb.policyMutex.Unlock()
	}
}

func (lb *LoadBalancer) OnEndpointsSynced() {
}

func (lb *LoadBalancer) OnDestinationRuleAdd(dr *istioapi.DestinationRule) {
	lb.lock.RLock()
	defer lb.lock.RUnlock()

	policyName := getPolicyName(dr)
	for svcPort, state := range lb.services {
		if !isNamespacedNameEqual(dr, &svcPort.NamespacedName) {
			continue
		}
		lb.policyMutex.Lock()
		lb.setLoadBalancerPolicy(dr, policyName, svcPort, state.endpoints)
		lb.policyMutex.Unlock()
	}
}

func (lb *LoadBalancer) OnDestinationRuleUpdate(oldDr, dr *istioapi.DestinationRule) {
	lb.lock.RLock()
	defer lb.lock.RUnlock()

	policyName := getPolicyName(dr)
	for svcPort, state := range lb.services {
		if !isNamespacedNameEqual(dr, &svcPort.NamespacedName) {
			continue
		}
		lb.policyMutex.Lock()
		if policy, exists := lb.policyMap[svcPort]; exists && policy.Name() != policyName {
			lb.policyMap[svcPort].Release()
			delete(lb.policyMap, svcPort)
		}
		lb.setLoadBalancerPolicy(dr, policyName, svcPort, state.endpoints)
		lb.policyMap[svcPort].Update(oldDr, dr)
		lb.policyMutex.Unlock()
	}
}

func (lb *LoadBalancer) OnDestinationRuleDelete(dr *istioapi.DestinationRule) {
	lb.lock.Lock()
	defer lb.lock.Unlock()

	for svcPort := range lb.services {
		if !isNamespacedNameEqual(dr, &svcPort.NamespacedName) {
			continue
		}
		lb.policyMutex.Lock()
		lb.policyMap[svcPort].Release()
		delete(lb.policyMap, svcPort)
		lb.policyMutex.Unlock()
	}
}

func (lb *LoadBalancer) OnDestinationRuleSynced() {
}

// setLoadBalancerPolicy new load-balance policy by policy name,
// this assumes that lb.policyMapLock is already held.
func (lb *LoadBalancer) setLoadBalancerPolicy(dr *istioapi.DestinationRule, policyName string, svcPort proxy.ServicePortName, endpoints []string) {
	switch policyName {
	case RoundRobin:
		lb.policyMap[svcPort] = NewRoundRobinPolicy()
	case Random:
		lb.policyMap[svcPort] = NewRandomPolicy()
	case ConsistentHash:
		lb.policyMap[svcPort] = NewConsistentHashPolicy(lb.Config.ConsistentHash, dr, endpoints)
	default:
		klog.Errorf("unsupported loadBalance policy %s", policyName)
		return
	}
}

// TryPickEndpoint try to pick a service endpoint from load-balance strategy.
func (lb *LoadBalancer) tryPickEndpoint(svcPort proxy.ServicePortName, sessionAffinityEnabled bool, endpoints []string,
	srcAddr net.Addr, netConn net.Conn, cliReq *http.Request) (string, *http.Request, bool) {
	lb.policyMutex.Lock()
	defer lb.policyMutex.Unlock()

	policy, exists := lb.policyMap[svcPort]
	if !exists {
		return "", cliReq, false
	}
	if exists && sessionAffinityEnabled {
		klog.Warningf("LoadBalancer policy conflicted with sessionAffinity: ClientIP")
		return "", cliReq, false
	}
	endpoint, req, err := policy.Pick(endpoints, srcAddr, netConn, cliReq)
	if err != nil {
		return "", req, false
	}
	return endpoint, req, true
}

func (lb *LoadBalancer) nextEndpointWithConn(svcPort proxy.ServicePortName, srcAddr net.Addr, sessionAffinityReset bool,
	netConn net.Conn, cliReq *http.Request) (string, *http.Request, error) {
	// Coarse locking is simple. We can get more fine-grained if/when we
	// can prove it matters.
	lb.lock.Lock()
	defer lb.lock.Unlock()

	state, exists := lb.services[svcPort]
	if !exists || state == nil {
		return "", cliReq, userspace.ErrMissingServiceEntry
	}
	if len(state.endpoints) == 0 {
		return "", cliReq, userspace.ErrMissingEndpoints
	}
	klog.V(4).InfoS("NextEndpoint for service", "servicePortName", svcPort, "address", srcAddr, "endpoints", state.endpoints)

	sessionAffinityEnabled := isSessionAffinity(&state.affinity)

	// Note: because loadBalance strategy may have read http.Request from inConn,
	// so here we need to return it to outConn!
	endpoint, req, picked := lb.tryPickEndpoint(svcPort, sessionAffinityEnabled, state.endpoints, srcAddr, netConn, cliReq)
	if picked {
		return endpoint, req, nil
	}

	var ipaddr string
	if sessionAffinityEnabled {
		// Caution: don't shadow ipaddr
		var err error
		ipaddr, _, err = net.SplitHostPort(srcAddr.String())
		if err != nil {
			return "", req, fmt.Errorf("malformed source address %q: %v", srcAddr.String(), err)
		}
		if !sessionAffinityReset {
			sessionAffinity, exists := state.affinity.affinityMap[ipaddr]
			if exists && int(time.Since(sessionAffinity.lastUsed).Seconds()) < state.affinity.ttlSeconds {
				// Affinity wins.
				endpoint := sessionAffinity.endpoint
				sessionAffinity.lastUsed = time.Now()
				klog.V(4).InfoS("NextEndpoint for service from IP with sessionAffinity", "servicePortName", svcPort, "IP", ipaddr, "sessionAffinity", sessionAffinity, "endpoint", endpoint)
				return endpoint, req, nil
			}
		}
	}
	// Take the next endpoint.
	endpoint = state.endpoints[state.index]
	state.index = (state.index + 1) % len(state.endpoints)

	if sessionAffinityEnabled {
		var affinity *affinityState
		affinity = state.affinity.affinityMap[ipaddr]
		if affinity == nil {
			affinity = new(affinityState) //&affinityState{ipaddr, "TCP", "", endpoint, time.Now()}
			state.affinity.affinityMap[ipaddr] = affinity
		}
		affinity.lastUsed = time.Now()
		affinity.endpoint = endpoint
		affinity.clientIP = ipaddr
		klog.V(4).InfoS("Updated affinity key", "IP", ipaddr, "affinityState", state.affinity.affinityMap[ipaddr])
	}

	return endpoint, req, nil
}

// TryConnectEndpoints attempts to connect to the next available endpoint for the given service, cycling
// through until it is able to successfully connect, or it has tried with all timeouts in EndpointDialTimeouts.
func (lb *LoadBalancer) TryConnectEndpoints(service proxy.ServicePortName, srcAddr net.Addr, protocol string,
	netConn net.Conn, cliReq *http.Request) (out net.Conn, err error) {
	sessionAffinityReset := false
	for _, dialTimeout := range userspace.EndpointDialTimeouts {
		endpoint, req, err := lb.nextEndpointWithConn(service, srcAddr, sessionAffinityReset, netConn, cliReq)
		if err != nil {
			klog.ErrorS(err, "Couldn't find an endpoint for service", "service", service)
			return nil, err
		}
		klog.V(3).InfoS("Mapped service to endpoint", "service", service, "endpoint", endpoint)
		// NOTE: outConn can be a net.Conn(golang) or network.Stream(libp2p)
		outConn, err := lb.dialEndpoint(protocol, endpoint)
		if err != nil {
			if netutil.IsTooManyFDsError(err) {
				panic("Dial failed: " + err.Error())
			}
			klog.ErrorS(err, "Dial failed")
			sessionAffinityReset = true
			time.Sleep(dialTimeout)
			continue
		}
		if req != nil {
			reqBytes, err := netutil.HttpRequestToBytes(req)
			if err == nil {
				_, err = outConn.Write(reqBytes)
				if err != nil {
					return nil, err
				}
			}
		}
		return outConn, nil
	}
	return nil, fmt.Errorf("failed to connect to an endpoint")
}

// dialEndpoint If the endpoint contains node name then try to dial stream conn or try to dial net conn.
func (lb *LoadBalancer) dialEndpoint(protocol, endpoint string) (net.Conn, error) {
	targetNode, targetPod, targetIP, targetPort, ok := parseEndpoint(endpoint)
	if !ok {
		return nil, fmt.Errorf("invalid endpoint %s", endpoint)
	}

	switch targetNode {
	case defaults.EmptyNodeName, lb.Config.NodeName:
		// TODO: This could spin up a new goroutine to make the outbound connection,
		// and keep accepting inbound traffic.
		outConn, err := net.Dial(protocol, net.JoinHostPort(targetIP, targetPort))
		if err != nil {
			return nil, err
		}
		klog.Infof("Dial legacy network between %s - {%s %s %s:%s}", targetPod, protocol, targetNode, targetIP, targetPort)
		return outConn, nil
	default:
		targetPort, err := strconv.ParseInt(targetPort, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid endpoint %s", endpoint)
		}
		proxyOpts := tunnel.ProxyOptions{Protocol: protocol, NodeName: targetNode, IP: targetIP, Port: int32(targetPort)}
		streamConn, err := tunnel.Agent.GetProxyStream(proxyOpts)
		if err != nil {
			return nil, fmt.Errorf("get proxy stream from %s error: %v", targetNode, err)
		}
		klog.Infof("Dial libp2p network between %s - {%s %s %s:%d}", targetPod, protocol, targetNode, targetIP, targetPort)
		return streamConn, nil
	}
}

func (lb *LoadBalancer) NextEndpoint(svcPort proxy.ServicePortName, srcAddr net.Addr, sessionAffinityReset bool) (string, error) {
	return "", nil
}

func (lb *LoadBalancer) NewService(svcPort proxy.ServicePortName, affinityType v1.ServiceAffinity, ttlSeconds int) error {
	klog.V(4).InfoS("NewService", "servicePortName", svcPort)
	lb.lock.Lock()
	defer lb.lock.Unlock()
	lb.newServiceInternal(svcPort, affinityType, ttlSeconds)
	return nil
}

// This assumes that lb.lock is already held.
func (lb *LoadBalancer) newServiceInternal(svcPort proxy.ServicePortName, affinityType v1.ServiceAffinity, ttlSeconds int) *balancerState {
	if ttlSeconds == 0 {
		ttlSeconds = int(v1.DefaultClientIPServiceAffinitySeconds) //default to 3 hours if not specified.  Should 0 be unlimited instead????
	}

	if _, exists := lb.services[svcPort]; !exists {
		lb.services[svcPort] = &balancerState{affinity: *newAffinityPolicy(affinityType, ttlSeconds)}
		klog.V(4).InfoS("service does not exist, created", "servicePortName", svcPort)
	} else if affinityType != "" {
		lb.services[svcPort].affinity.affinityType = affinityType
	}
	return lb.services[svcPort]
}

func (lb *LoadBalancer) DeleteService(svcPort proxy.ServicePortName) {
	klog.V(4).InfoS("DeleteService", "servicePortName", svcPort)
	lb.lock.Lock()
	defer lb.lock.Unlock()
	delete(lb.services, svcPort)
}

func (lb *LoadBalancer) CleanupStaleStickySessions(svcPort proxy.ServicePortName) {
	lb.lock.Lock()
	defer lb.lock.Unlock()

	state, exists := lb.services[svcPort]
	if !exists {
		return
	}
	for ip, affinity := range state.affinity.affinityMap {
		if int(time.Since(affinity.lastUsed).Seconds()) >= state.affinity.ttlSeconds {
			klog.V(4).InfoS("Removing client from affinityMap for service", "IP", affinity.clientIP, "servicePortName", svcPort)
			delete(state.affinity.affinityMap, ip)
		}
	}
}

func (lb *LoadBalancer) ServiceHasEndpoints(svcPort proxy.ServicePortName) bool {
	lb.lock.RLock()
	defer lb.lock.RUnlock()
	state, exists := lb.services[svcPort]
	// TODO: while nothing ever assigns nil to the map, *some* of the code using the map
	// checks for it.  The code should all follow the same convention.
	return exists && state != nil && len(state.endpoints) > 0
}

func (lb *LoadBalancer) GetServicePortName(namespacedName types.NamespacedName, port int) (proxy.ServicePortName, bool) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	for svcPort, so := range lb.serviceMap {
		if svcPort.NamespacedName == namespacedName && so.port == port {
			return svcPort, true
		}
	}
	return proxy.ServicePortName{}, false
}
