package loadbalancer

import (
	"fmt"
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
	servicehelper "k8s.io/cloud-provider/service/helpers"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/proxy"
	"k8s.io/kubernetes/pkg/proxy/userspace"
	utilproxy "k8s.io/kubernetes/pkg/proxy/util"
	"k8s.io/kubernetes/pkg/util/async"
	netutils "k8s.io/utils/net"
	stringslices "k8s.io/utils/strings/slices"
	"math/rand"
	"net"
	"net/http"
	"reflect"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kubeedge/edgemesh/pkg/apis/config/defaults"
	"github.com/kubeedge/edgemesh/pkg/apis/config/v1alpha1"
	"github.com/kubeedge/edgemesh/pkg/tunnel"
	netutil "github.com/kubeedge/edgemesh/pkg/util/net"
)

type portal struct {
	ip         net.IP
	port       int
	isExternal bool
}

// ServiceInfo contains information and state for a particular proxied service
type ServiceInfo struct {
	isAliveAtomic       int32 // Only access this with atomic ops
	portal              portal
	protocol            v1.Protocol
	proxyPort           int
	nodePort            int
	loadBalancerStatus  v1.LoadBalancerStatus
	sessionAffinityType v1.ServiceAffinity
	stickyMaxAgeSeconds int
	// Deprecated, but required for back-compat (including e2e)
	externalIPs []string

	// isStartedAtomic is set to non-zero when the service's socket begins
	// accepting requests. Used in testcases. Only access this with atomic ops.
	isStartedAtomic int32
	// isFinishedAtomic is set to non-zero when the service's socket shuts
	// down. Used in testcases. Only access this with atomic ops.
	isFinishedAtomic int32
}

func (info *ServiceInfo) setStarted() {
	atomic.StoreInt32(&info.isStartedAtomic, 1)
}

func (info *ServiceInfo) IsStarted() bool {
	return atomic.LoadInt32(&info.isStartedAtomic) != 0
}

func (info *ServiceInfo) setFinished() {
	atomic.StoreInt32(&info.isFinishedAtomic, 1)
}

func (info *ServiceInfo) IsFinished() bool {
	return atomic.LoadInt32(&info.isFinishedAtomic) != 0
}

func (info *ServiceInfo) setAlive(b bool) {
	var i int32
	if b {
		i = 1
	}
	atomic.StoreInt32(&info.isAliveAtomic, i)
}

func (info *ServiceInfo) IsAlive() bool {
	return atomic.LoadInt32(&info.isAliveAtomic) != 0
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
	endpoints        []string          // a list of "nodeName:podName:ip:port" style strings, nodeName and podName can be empty!!!
	noReadyEndpoints map[string]string //不可用节点ep信息  key -->nodeName:podName:ip:port value-->0
	index            int               // current index into endpoints
	affinity         affinityPolicy
}

const numBurstSyncs int = 2

type serviceChange struct {
	current  *v1.Service
	previous *v1.Service
}

// Interface for async runner; abstracted for testing
type asyncRunnerInterface interface {
	Run()
	Loop(<-chan struct{})
}

// assert LoadBalancer is an userspace.LoadBalancer
var _ userspace.LoadBalancer = &LoadBalancer{}

type LoadBalancer struct {
	Config      *v1alpha1.LoadBalancer
	kubeClient  kubernetes.Interface
	istioClient istio.Interface
	syncPeriod  time.Duration
	mu          sync.Mutex // protects serviceMap
	serviceMap  map[proxy.ServicePortName]*ServiceInfo
	lock        sync.RWMutex // protects services
	services    map[proxy.ServicePortName]*balancerState
	policyMutex sync.Mutex // protects policyMap
	policyMap   map[proxy.ServicePortName]Policy
	stopCh      chan struct{}
	// endpointsSynced and servicesSynced are set to 1 when the corresponding
	// objects are synced after startup. This is used to avoid updating iptables
	// with some partial data after kube-proxy restart.
	endpointsSynced int32
	servicesSynced  int32
	initialized     int32
	// protects serviceChanges
	serviceChangesLock sync.Mutex
	serviceChanges     map[types.NamespacedName]*serviceChange // map of service changes
	syncRunner         asyncRunnerInterface                    // governs calls to syncProxyRules
}

func New(config *v1alpha1.LoadBalancer, kubeClient kubernetes.Interface, istioClient istio.Interface, syncPeriod time.Duration) *LoadBalancer {
	lb := &LoadBalancer{
		Config:         config,
		kubeClient:     kubeClient,
		istioClient:    istioClient,
		syncPeriod:     syncPeriod,
		serviceMap:     make(map[proxy.ServicePortName]*ServiceInfo),
		serviceChanges: make(map[types.NamespacedName]*serviceChange),
		services:       make(map[proxy.ServicePortName]*balancerState),
		policyMap:      make(map[proxy.ServicePortName]Policy),
		stopCh:         make(chan struct{}),
	}
	lb.syncRunner = async.NewBoundedFrequencyRunner("sync-runner", lb.syncServices, time.Minute, syncPeriod, numBurstSyncs)
	return lb
}

func (lb *LoadBalancer) isInitialized() bool {
	return atomic.LoadInt32(&lb.initialized) > 0
}

func (lb *LoadBalancer) syncServices() {
	start := time.Now()
	defer func() {
		klog.V(4).InfoS("syncServices complete", "elapsed", time.Since(start))
	}()

	// don't sync rules till we've received services and endpoints
	if !lb.isInitialized() {
		klog.V(2).InfoS("Not syncing userspace proxy until Services and Endpoints have been received from master")
		return
	}

	lb.serviceChangesLock.Lock()
	changes := lb.serviceChanges
	lb.serviceChanges = make(map[types.NamespacedName]*serviceChange)
	lb.serviceChangesLock.Unlock()

	lb.mu.Lock()
	defer lb.mu.Unlock()

	klog.V(4).InfoS("userspace proxy: processing service events", "count", len(changes))
	for _, change := range changes {
		existingPorts := lb.mergeService(change.current)
		lb.unmergeService(change.previous, existingPorts)
	}

	lb.cleanupStaleStickySessions()
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
		go lb.syncRunner.Loop(lb.stopCh)
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

func sameConfig(info *ServiceInfo, service *v1.Service, port *v1.ServicePort) bool {
	if info.protocol != port.Protocol || info.portal.port != int(port.Port) || info.nodePort != int(port.NodePort) {
		return false
	}
	if !info.portal.ip.Equal(netutils.ParseIPSloppy(service.Spec.ClusterIP)) {
		return false
	}
	if !ipsEqual(info.externalIPs, service.Spec.ExternalIPs) {
		return false
	}
	if !servicehelper.LoadBalancerStatusEqual(&info.loadBalancerStatus, &service.Status.LoadBalancer) {
		return false
	}
	if info.sessionAffinityType != service.Spec.SessionAffinity {
		return false
	}
	return true
}

func ipsEqual(lhs, rhs []string) bool {
	if len(lhs) != len(rhs) {
		return false
	}
	for i := range lhs {
		if lhs[i] != rhs[i] {
			return false
		}
	}
	return true
}

func (lb *LoadBalancer) mergeService(service *v1.Service) sets.String {
	if service == nil {
		return nil
	}
	if utilproxy.ShouldSkipService(service) {
		return nil
	}
	existingPorts := sets.NewString()
	svcName := types.NamespacedName{Namespace: service.Namespace, Name: service.Name}
	for i := range service.Spec.Ports {
		servicePort := &service.Spec.Ports[i]
		serviceName := proxy.ServicePortName{NamespacedName: svcName, Port: servicePort.Name}
		existingPorts.Insert(servicePort.Name)
		info, exists := lb.serviceMap[serviceName]
		// TODO: check health of the socket? What if ProxyLoop exited?
		if exists && sameConfig(info, service, servicePort) {
			// Nothing changed.
			continue
		}
		if exists {
			klog.V(4).InfoS("Something changed for service: stopping it", "serviceName", serviceName)
			delete(lb.serviceMap, serviceName)
			info.setFinished()
		}

		serviceIP := netutils.ParseIPSloppy(service.Spec.ClusterIP)
		klog.V(1).InfoS("Adding new service", "serviceName", serviceName, "addr", net.JoinHostPort(serviceIP.String(), strconv.Itoa(int(servicePort.Port))), "protocol", servicePort.Protocol)
		info = &ServiceInfo{
			isAliveAtomic:       1,
			protocol:            servicePort.Protocol,
			sessionAffinityType: v1.ServiceAffinityNone,
			portal: portal{
				ip:   serviceIP,
				port: int(servicePort.Port),
			},
			externalIPs: service.Spec.ExternalIPs,
		}
		// Deep-copy in case the service instance changes
		info.loadBalancerStatus = *service.Status.LoadBalancer.DeepCopy()
		info.nodePort = int(servicePort.NodePort)
		info.sessionAffinityType = service.Spec.SessionAffinity
		lb.serviceMap[serviceName] = info
		// Kube-apiserver side guarantees SessionAffinityConfig won't be nil when session affinity type is ClientIP
		if service.Spec.SessionAffinity == v1.ServiceAffinityClientIP {
			info.stickyMaxAgeSeconds = int(*service.Spec.SessionAffinityConfig.ClientIP.TimeoutSeconds)
		}

		klog.V(4).InfoS("Record serviceInfo", "serviceInfo", info)

		if err := lb.NewService(serviceName, info.sessionAffinityType, info.stickyMaxAgeSeconds); err != nil {
			klog.ErrorS(err, "Failed to new service", "serviceName", serviceName)
		}

		info.setStarted()
	}

	return existingPorts
}

func (lb *LoadBalancer) unmergeService(service *v1.Service, existingPorts sets.String) {
	if service == nil {
		return
	}

	if utilproxy.ShouldSkipService(service) {
		return
	}
	staleUDPServices := sets.NewString()
	svcName := types.NamespacedName{Namespace: service.Namespace, Name: service.Name}
	for i := range service.Spec.Ports {
		servicePort := &service.Spec.Ports[i]
		if existingPorts.Has(servicePort.Name) {
			continue
		}
		serviceName := proxy.ServicePortName{NamespacedName: svcName, Port: servicePort.Name}

		klog.V(1).InfoS("Stopping service", "serviceName", serviceName)
		info, exists := lb.serviceMap[serviceName]
		if !exists {
			klog.ErrorS(nil, "Service is being removed but doesn't exist", "serviceName", serviceName)
			continue
		}

		if lb.serviceMap[serviceName].protocol == v1.ProtocolUDP {
			staleUDPServices.Insert(lb.serviceMap[serviceName].portal.ip.String())
		}

		delete(lb.serviceMap, serviceName)
		lb.DeleteService(serviceName)
		info.setFinished()
	}
}

func (lb *LoadBalancer) serviceChange(previous, current *v1.Service, detail string) {
	var svcName types.NamespacedName
	if current != nil {
		svcName = types.NamespacedName{Namespace: current.Namespace, Name: current.Name}
	} else {
		svcName = types.NamespacedName{Namespace: previous.Namespace, Name: previous.Name}
	}
	klog.V(4).InfoS("Record service change", "action", detail, "svcName", svcName)

	lb.serviceChangesLock.Lock()
	defer lb.serviceChangesLock.Unlock()

	change, exists := lb.serviceChanges[svcName]
	if !exists {
		// change.previous is only set for new changes. We must keep
		// the oldest service info (or nil) because correct unmerging
		// depends on the next update/del after a merge, not subsequent
		// updates.
		change = &serviceChange{previous: previous}
		lb.serviceChanges[svcName] = change
	}

	// Always use the most current service (or nil) as change.current
	change.current = current

	if reflect.DeepEqual(change.previous, change.current) {
		// collapsed change had no effect
		delete(lb.serviceChanges, svcName)
	} else if lb.isInitialized() {
		// change will have an effect, ask the proxy to sync
		lb.syncRunner.Run()
	}
}

func (lb *LoadBalancer) OnServiceAdd(service *v1.Service) {
	lb.serviceChange(nil, service, "OnServiceAdd")
}

func (lb *LoadBalancer) OnServiceUpdate(oldService, service *v1.Service) {
	lb.serviceChange(oldService, service, "OnServiceUpdate")
	klog.Info("")
}

func (lb *LoadBalancer) OnServiceDelete(service *v1.Service) {
	lb.serviceChange(service, nil, "OnServiceDelete")
}

func (lb *LoadBalancer) OnServiceSynced() {
	klog.V(2).InfoS("LoadBalancer OnServiceSynced")

	// Mark services as initialized and (if endpoints are already
	// initialized) the entire proxy as initialized
	atomic.StoreInt32(&lb.servicesSynced, 1)
	if atomic.LoadInt32(&lb.endpointsSynced) > 0 {
		atomic.StoreInt32(&lb.initialized, 1)
	}

	// Must sync from a goroutine to avoid blocking the
	// service event handler on startup with large numbers
	// of initial objects
	go lb.syncServices()
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
			state.endpoints = utilproxy.ShuffleStrings(newEndpoints[0])
			noReadyEp := utilproxy.ShuffleStrings(newEndpoints[1])
			if len(state.noReadyEndpoints) == 0 {
				state.noReadyEndpoints = map[string]string{}
			}
			for _, ep := range noReadyEp {
				state.noReadyEndpoints[ep] = ""
			}
			klog.Infof("[OnEndpointsAdd](初始化)服务信息:%s,在线端点信息:[%s],离线端点信息:[%s]", svcPort, state.endpoints, state.noReadyEndpoints)
			// Reset the round-robin index.
			state.index = 0

			// Sync the backend loadBalancer policy.
			lb.policyMutex.Lock()
			if policy, exists := lb.policyMap[svcPort]; exists {
				policy.Sync(newEndpoints[0])
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
		curNoReadyEndpoints := make(map[string]string)
		if state != nil {
			curNoReadyEndpoints = state.noReadyEndpoints
		}
		klog.Infof("[OnEndpointsUpdate]:更新中 endpoints for service,servicePortName:%s,ready-endpoints:%s,noReady-endpoints:%s,当前ready-endpoints:%s,当前noReady-endpoints:%s", svcPort, newEndpoints[0], newEndpoints[1], curEndpoints, curNoReadyEndpoints)
		if !exists || state == nil || len(curEndpoints) != len(newEndpoints[0]) || !slicesEquiv(stringslices.Clone(curEndpoints), newEndpoints[0]) || !reflect.DeepEqual(curNoReadyEndpoints, newEndpoints[1]) {
			klog.V(1).InfoS("LoadBalancerEX: Setting endpoints for service", "servicePortName", svcPort, "endpoints", newEndpoints)
			lb.removeStaleAffinity(svcPort, newEndpoints[0])
			// OnEndpointsUpdate can be called without NewService being called externally.
			// To be safe we will call it here.  A new service will only be created
			// if one does not already exist.  The affinity will be updated
			// later, once NewService is called.
			state = lb.newServiceInternal(svcPort, v1.ServiceAffinity(""), 0)
			state.endpoints = utilproxy.ShuffleStrings(newEndpoints[0])
			noReadyEp := utilproxy.ShuffleStrings(newEndpoints[1])
			state.noReadyEndpoints = map[string]string{}
			for _, ep := range noReadyEp {
				state.noReadyEndpoints[ep] = ""
			}
			// Reset the round-robin index.
			state.index = 0

			// Sync the backend loadBalancer policy.
			lb.policyMutex.Lock()
			if policy, exists := lb.policyMap[svcPort]; exists {
				policy.Sync(newEndpoints[0])
			}
			lb.policyMutex.Unlock()
		}
		registeredEndpoints[svcPort] = true
		klog.Infof("[OnEndpointsUpdate](已更新)服务信息:%s,在线端点信息:[%s],离线端点信息:[%s]", svcPort, state.endpoints, state.noReadyEndpoints)
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
		if len(state.noReadyEndpoints) > 0 {
			klog.V(2).InfoS("LoadBalancerEX: Removing endpoints service", "servicePortName", svcPort)
			state.noReadyEndpoints = map[string]string{}
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
	klog.V(2).InfoS(" OnEndpointsSynced")

	// Mark endpoints as initialized and (if services are already
	// initialized) the entire proxy as initialized
	atomic.StoreInt32(&lb.endpointsSynced, 1)
	if atomic.LoadInt32(&lb.servicesSynced) > 0 {
		atomic.StoreInt32(&lb.initialized, 1)
	}

	// Must sync from a goroutine to avoid blocking the
	// service event handler on startup with large numbers
	// of initial objects
	go lb.syncServices()
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
	klog.Infof("[TryPickEndpoint]服务信息:%s,策略：%s", svcPort, policy.Name())
	if exists && sessionAffinityEnabled {
		klog.Warningf("LoadBalancer policy conflicted with sessionAffinity: ClientIP")
		return "", cliReq, false
	}
	endpoint, req, err := policy.Pick(endpoints, srcAddr, netConn, cliReq)
	if err != nil {
		klog.Warningf("[TryPickEndpoint]服务信息:%s,失败原因：%s", svcPort, err)
		return "", req, false
	}
	klog.Infof("[TryPickEndpoint]服务信息:%s,结果：%s", svcPort, endpoint)
	return endpoint, req, true
}

// TryPickNoReadyEndpoint try to pick a service endpoint from load-balance strategy.
func (lb *LoadBalancer) TryPickNoReadyEndpoint(svcPort proxy.ServicePortName, endpoints []string,
	srcAddr net.Addr, netConn net.Conn, cliReq *http.Request) (string, *http.Request, bool) {
	lb.policyMutex.Lock()
	defer lb.policyMutex.Unlock()

	policy, exists := lb.policyMap[svcPort]
	if !exists {
		return "", cliReq, false
	}
	klog.Infof("[TryPickNoReadyEndpoint]服务信息:%s,策略：%s", svcPort, policy.Name())
	endpoint, req, err := policy.Pick(endpoints, srcAddr, netConn, cliReq)
	if err != nil {
		klog.Warningf("[TryPickNoReadyEndpoint]服务信息:%s,失败原因：%s", svcPort, err)
		return "", req, false
	}
	klog.Infof("[TryPickNoReadyEndpoint]服务信息:%s,结果：%s", svcPort, endpoint)
	return endpoint, req, true
}

func (lb *LoadBalancer) nextEndpointWithConn(svcPort proxy.ServicePortName, srcAddr net.Addr, sessionAffinityReset bool,
	netConn net.Conn, cliReq *http.Request, delNoReadyEndpoint []string) (string, *http.Request, error) {
	// Coarse locking is simple. We can get more fine-grained if/when we
	// can prove it matters.
	lb.lock.Lock()
	defer lb.lock.Unlock()

	state, exists := lb.services[svcPort]
	if !exists || state == nil {
		return "", cliReq, userspace.ErrMissingServiceEntry
	}
	klog.Infof("[NextEndpoint](未清理)服务信息:%s,在线端点信息:[%s],离线端点信息:[%s]", svcPort, state.endpoints, state.noReadyEndpoints)
	//移除已经验证访问过的不可用节点的ep信息
	if len(delNoReadyEndpoint) != 0 && delNoReadyEndpoint[0] != "" {
		delete(state.noReadyEndpoints, delNoReadyEndpoint[0])
	}
	//组装剩余不可用节点的ep信息
	var noReadyEndpoints []string
	if len(state.noReadyEndpoints) != 0 {
		for ep, _ := range state.noReadyEndpoints {
			noReadyEndpoints = append(noReadyEndpoints, ep)
		}
	}
	klog.Infof("[NextEndpoint](已更新)服务信息:%s,客户端信息:[%s],在线端点信息:[%s],离线端点信息:[%s]", svcPort, srcAddr.String(), state.endpoints, noReadyEndpoints)
	if len(state.endpoints) == 0 {
		//所有ready节点都没有时使用noready节点负载
		if len(noReadyEndpoints) != 0 {
			// Take the next endpoint.
			endpoint, req, picked := lb.TryPickNoReadyEndpoint(svcPort, noReadyEndpoints, srcAddr, netConn, cliReq)
			klog.Infof("[NextEndpoint]noReady endpoint选址,服务信息:%s,结果：%s,picked:%s", svcPort, endpoint, picked)
			if picked {
				//如果时noReady节点访问异常之后，自动清理对应的endpoints信息
				delNoReadyEndpoint[0] = endpoint
				return endpoint, req, nil
			} else {
				//策略选择失败，则默认使用随机选择
				k := rand.Int() % len(noReadyEndpoints)
				//如果时noReady节点访问异常之后，自动清理对应的endpoints信息
				delNoReadyEndpoint[0] = noReadyEndpoints[k]
				return noReadyEndpoints[k], req, nil
			}
		}
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
	delNoReadyEndpoint := make([]string, 1)
	for _, dialTimeout := range userspace.EndpointDialTimeouts {
		endpoint, req, err := lb.nextEndpointWithConn(service, srcAddr, sessionAffinityReset, netConn, cliReq, delNoReadyEndpoint)
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
			klog.ErrorS(err, "Dial failed", "endpoint:", endpoint)
			sessionAffinityReset = true
			time.Sleep(dialTimeout)
			continue
		}
		if req != nil {
			reqBytes, err := netutil.HTTPRequestToBytes(req)
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

	for svcPort, info := range lb.serviceMap {
		if svcPort.NamespacedName == namespacedName && info.portal.port == port {
			return svcPort, true
		}
	}
	return proxy.ServicePortName{}, false
}
