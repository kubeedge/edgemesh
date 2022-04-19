// This package is copied from Kubernetes project.
// https://github.com/kubernetes/kubernetes/blob/v1.23.0/pkg/proxy/userspace/roundrobin.go
// Use LoadBalancerEX to provide richer load balancing.
package userspace

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"sort"
	"strconv"
	"sync"
	"time"

	istioapi "istio.io/client-go/pkg/apis/networking/v1alpha3"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/proxy"
	"k8s.io/kubernetes/pkg/proxy/util"
	stringslices "k8s.io/utils/strings/slices"
)

var (
	ErrMissingServiceEntry = errors.New("missing service entry")
	ErrMissingEndpoints    = errors.New("missing endpoints")
)

type affinityState struct {
	clientIP string
	//clientProtocol  api.Protocol //not yet used
	//sessionCookie   string       //not yet used
	endpoint string
	lastUsed time.Time
}

type affinityPolicy struct {
	affinityType v1.ServiceAffinity
	affinityMap  map[string]*affinityState // map client IP -> affinity info
	ttlSeconds   int
}

// LoadBalancerEX is a extended load balancer.
type LoadBalancerEX struct {
	lock        sync.RWMutex
	services    map[proxy.ServicePortName]*balancerState
	strategyMap map[proxy.ServicePortName]LoadBalancerStrategy
}

// Ensure this implements LoadBalancer.
var _ LoadBalancer = &LoadBalancerEX{}

type balancerState struct {
	endpoints []string // a list of "nodeName:podName:ip:port" style strings, nodeName and podName can be empty!!!
	index     int      // current index into endpoints
	affinity  affinityPolicy
}

// isValidEndpoint checks that the given host / port pair are valid endpoint
func isValidEndpoint(host string, port int) bool {
	return host != "" && port > 0
}

// buildPortsToEndpointsMap builds a map of portname -> all nodeName:podName:ip:port
// for that portname. Explode Endpoints.Subsets[*] into this structure.
func buildPortsToEndpointsMap(endpoints *v1.Endpoints) map[string][]string {
	portsToEndpoints := map[string][]string{}
	for i := range endpoints.Subsets {
		ss := &endpoints.Subsets[i]
		for i := range ss.Ports {
			port := &ss.Ports[i]
			for i := range ss.Addresses {
				addr := &ss.Addresses[i]
				if isValidEndpoint(addr.IP, int(port.Port)) {
					nodeName := EmptyNodeName
					podName := EmptyPodName
					if addr.NodeName != nil {
						nodeName = *addr.NodeName
					}
					if addr.TargetRef != nil {
						podName = addr.TargetRef.Name
					}
					endpoint := fmt.Sprintf("%s:%s:%s", nodeName, podName, net.JoinHostPort(addr.IP, strconv.Itoa(int(port.Port))))
					portsToEndpoints[port.Name] = append(portsToEndpoints[port.Name], endpoint)
				}
			}
		}
	}
	return portsToEndpoints
}

func newAffinityPolicy(affinityType v1.ServiceAffinity, ttlSeconds int) *affinityPolicy {
	return &affinityPolicy{
		affinityType: affinityType,
		affinityMap:  make(map[string]*affinityState),
		ttlSeconds:   ttlSeconds,
	}
}

// NewLoadBalancerEX returns a new LoadBalancerEX.
func NewLoadBalancerEX() *LoadBalancerEX {
	return &LoadBalancerEX{
		services:    map[proxy.ServicePortName]*balancerState{},
		strategyMap: map[proxy.ServicePortName]LoadBalancerStrategy{},
	}
}

func (lb *LoadBalancerEX) NewService(svcPort proxy.ServicePortName, affinityType v1.ServiceAffinity, ttlSeconds int) error {
	klog.V(4).InfoS("LoadBalancerEX NewService", "servicePortName", svcPort)
	lb.lock.Lock()
	defer lb.lock.Unlock()
	lb.newServiceInternal(svcPort, affinityType, ttlSeconds)
	return nil
}

// This assumes that lb.lock is already held.
func (lb *LoadBalancerEX) newServiceInternal(svcPort proxy.ServicePortName, affinityType v1.ServiceAffinity, ttlSeconds int) *balancerState {
	if ttlSeconds == 0 {
		ttlSeconds = int(v1.DefaultClientIPServiceAffinitySeconds) //default to 3 hours if not specified.  Should 0 be unlimited instead????
	}

	if _, exists := lb.services[svcPort]; !exists {
		lb.services[svcPort] = &balancerState{affinity: *newAffinityPolicy(affinityType, ttlSeconds)}
		klog.V(4).InfoS("LoadBalancerEX service does not exist, created", "servicePortName", svcPort)
	} else if affinityType != "" {
		lb.services[svcPort].affinity.affinityType = affinityType
	}
	return lb.services[svcPort]
}

func (lb *LoadBalancerEX) DeleteService(svcPort proxy.ServicePortName) {
	klog.V(4).InfoS("LoadBalancerEX DeleteService", "servicePortName", svcPort)
	lb.lock.Lock()
	defer lb.lock.Unlock()
	delete(lb.services, svcPort)
}

// return true if this service is using some form of session affinity.
func isSessionAffinity(affinity *affinityPolicy) bool {
	// Should never be empty string, but checking for it to be safe.
	if affinity.affinityType == "" || affinity.affinityType == v1.ServiceAffinityNone {
		return false
	}
	return true
}

// ServiceHasEndpoints checks whether a service entry has endpoints.
func (lb *LoadBalancerEX) ServiceHasEndpoints(svcPort proxy.ServicePortName) bool {
	lb.lock.RLock()
	defer lb.lock.RUnlock()
	state, exists := lb.services[svcPort]
	// TODO: while nothing ever assigns nil to the map, *some* of the code using the map
	// checks for it.  The code should all follow the same convention.
	return exists && state != nil && len(state.endpoints) > 0
}

// TryPickEndpoint try to pick a service endpoint from load-balance strategy.
func (lb *LoadBalancerEX) TryPickEndpoint(svcPort proxy.ServicePortName, sessionAffinityEnabled bool, endpoints []string, srcAddr net.Addr, netConn net.Conn) (string, *http.Request, bool) {
	strategy, exists := lb.strategyMap[svcPort]
	if !exists {
		return "", nil, false
	}
	if exists && sessionAffinityEnabled {
		klog.Warningf("LoadBalancer strategy conflicted with sessionAffinity: ClientIP")
		return "", nil, false
	}
	endpoint, req, err := strategy.Pick(endpoints, srcAddr, netConn)
	if err != nil {
		return "", req, false
	}
	return endpoint, req, true
}

// NextEndpoint returns a service endpoint.
func (lb *LoadBalancerEX) NextEndpoint(svcPort proxy.ServicePortName, srcAddr net.Addr, netConn net.Conn, sessionAffinityReset bool) (string, *http.Request, error) {
	// Coarse locking is simple.  We can get more fine-grained if/when we
	// can prove it matters.
	lb.lock.Lock()
	defer lb.lock.Unlock()

	state, exists := lb.services[svcPort]
	if !exists || state == nil {
		return "", nil, ErrMissingServiceEntry
	}
	if len(state.endpoints) == 0 {
		return "", nil, ErrMissingEndpoints
	}
	klog.V(4).InfoS("NextEndpoint for service", "servicePortName", svcPort, "address", srcAddr, "endpoints", state.endpoints)

	sessionAffinityEnabled := isSessionAffinity(&state.affinity)

	// Note: because load-balance strategy may have read http.Request from inConn,
	// so here we need to return it to outConn!
	endpoint, req, picked := lb.TryPickEndpoint(svcPort, sessionAffinityEnabled, state.endpoints, srcAddr, netConn)
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

// Remove any session affinity records associated to a particular endpoint (for example when a pod goes down).
func removeSessionAffinityByEndpoint(state *balancerState, svcPort proxy.ServicePortName, endpoint string) {
	for _, affinity := range state.affinity.affinityMap {
		if affinity.endpoint == endpoint {
			klog.V(4).InfoS("Removing client from affinityMap for service", "endpoint", affinity.endpoint, "servicePortName", svcPort)
			delete(state.affinity.affinityMap, affinity.clientIP)
		}
	}
}

// Loop through the valid endpoints and then the endpoints associated with the Load Balancer.
// Then remove any session affinity records that are not in both lists.
// This assumes the lb.lock is held.
func (lb *LoadBalancerEX) removeStaleAffinity(svcPort proxy.ServicePortName, newEndpoints []string) {
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

func (lb *LoadBalancerEX) OnEndpointsAdd(endpoints *v1.Endpoints) {
	portsToEndpoints := buildPortsToEndpointsMap(endpoints)

	lb.lock.Lock()
	defer lb.lock.Unlock()

	svcName := types.NamespacedName{Namespace: endpoints.Namespace, Name: endpoints.Name}
	for portname := range portsToEndpoints {
		svcPort := proxy.ServicePortName{NamespacedName: svcName, Port: portname}
		newEndpoints := portsToEndpoints[portname]
		state, exists := lb.services[svcPort]

		if !exists || state == nil || len(newEndpoints) > 0 {
			klog.V(1).InfoS("LoadBalancerEX: Setting endpoints service", "servicePortName", svcPort, "endpoints", newEndpoints)
			// OnEndpointsAdd can be called without NewService being called externally.
			// To be safe we will call it here.  A new service will only be created
			// if one does not already exist.
			state = lb.newServiceInternal(svcPort, v1.ServiceAffinity(""), 0)
			state.endpoints = util.ShuffleStrings(newEndpoints)

			// Reset the round-robin index.
			state.index = 0

			// Sync the load-balance strategy.
			if _, exists := lb.strategyMap[svcPort]; exists {
				lb.strategyMap[svcPort].Sync(newEndpoints)
			}
		}
	}
}

func (lb *LoadBalancerEX) OnEndpointsUpdate(oldEndpoints, endpoints *v1.Endpoints) {
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
			state.endpoints = util.ShuffleStrings(newEndpoints)

			// Reset the round-robin index.
			state.index = 0

			// Sync the load-balance strategy.
			if _, exists := lb.strategyMap[svcPort]; exists {
				lb.strategyMap[svcPort].Sync(newEndpoints)
			}
		}
		registeredEndpoints[svcPort] = true
	}

	// Now remove all endpoints missing from the update.
	for portname := range oldPortsToEndpoints {
		svcPort := proxy.ServicePortName{NamespacedName: svcName, Port: portname}
		if _, exists := registeredEndpoints[svcPort]; !exists {
			lb.resetService(svcPort)

			// Sync the load-balance strategy.
			if _, exists := lb.strategyMap[svcPort]; exists {
				lb.strategyMap[svcPort].Sync(nil)
			}
		}
	}
}

func (lb *LoadBalancerEX) resetService(svcPort proxy.ServicePortName) {
	// If the service is still around, reset but don't delete.
	if state, ok := lb.services[svcPort]; ok {
		if len(state.endpoints) > 0 {
			klog.V(2).InfoS("LoadBalancerEX: Removing endpoints service", "servicePortName", svcPort)
			state.endpoints = []string{}
		}
		state.index = 0
		state.affinity.affinityMap = map[string]*affinityState{}
	}
}

func (lb *LoadBalancerEX) OnEndpointsDelete(endpoints *v1.Endpoints) {
	portsToEndpoints := buildPortsToEndpointsMap(endpoints)

	lb.lock.Lock()
	defer lb.lock.Unlock()

	for portname := range portsToEndpoints {
		svcPort := proxy.ServicePortName{NamespacedName: types.NamespacedName{Namespace: endpoints.Namespace, Name: endpoints.Name}, Port: portname}
		lb.resetService(svcPort)

		// Sync the load-balance strategy.
		if _, exists := lb.strategyMap[svcPort]; exists {
			lb.strategyMap[svcPort].Sync(nil)
		}
	}
}

func (lb *LoadBalancerEX) OnEndpointsSynced() {
}

// Tests whether two slices are equivalent.  This sorts both slices in-place.
func slicesEquiv(lhs, rhs []string) bool {
	if len(lhs) != len(rhs) {
		return false
	}
	sort.Strings(lhs)
	sort.Strings(rhs)
	return stringslices.Equal(lhs, rhs)
}

func (lb *LoadBalancerEX) CleanupStaleStickySessions(svcPort proxy.ServicePortName) {
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

func (lb *LoadBalancerEX) OnDestinationRuleAdd(dr *istioapi.DestinationRule) {
	lb.lock.Lock()
	defer lb.lock.Unlock()

	strategyName := getStrategyName(dr)
	for svcPort, state := range lb.services {
		if !isNamespacedNameEqual(dr, &svcPort.NamespacedName) {
			continue
		}
		lb.setLoadBalancerStrategy(dr, strategyName, svcPort, state.endpoints)
	}
}

func (lb *LoadBalancerEX) OnDestinationRuleUpdate(oldDr, dr *istioapi.DestinationRule) {
	lb.lock.Lock()
	defer lb.lock.Unlock()
	strategyName := getStrategyName(dr)
	for svcPort, state := range lb.services {
		if !isNamespacedNameEqual(dr, &svcPort.NamespacedName) {
			continue
		}
		if _, exists := lb.strategyMap[svcPort]; !exists {
			lb.setLoadBalancerStrategy(dr, strategyName, svcPort, state.endpoints)
			lb.strategyMap[svcPort].Update(oldDr, dr)
		} else if strategyName != "" && lb.strategyMap[svcPort].Name() != strategyName {
			lb.strategyMap[svcPort].Release()                                      // release old
			lb.setLoadBalancerStrategy(dr, strategyName, svcPort, state.endpoints) // set new
			lb.strategyMap[svcPort].Update(oldDr, dr)
		}
	}
}

func (lb *LoadBalancerEX) OnDestinationRuleDelete(dr *istioapi.DestinationRule) {
	lb.lock.Lock()
	defer lb.lock.Unlock()

	for svcPort := range lb.services {
		if !isNamespacedNameEqual(dr, &svcPort.NamespacedName) {
			continue
		}
		lb.strategyMap[svcPort].Release()
		delete(lb.strategyMap, svcPort)
	}
}

func (lb *LoadBalancerEX) OnDestinationRuleSynced() {
}

// setLoadBalancerStrategy new load-balance strategy by strategy name,
// this assumes that lb.lock is already held.
func (lb *LoadBalancerEX) setLoadBalancerStrategy(dr *istioapi.DestinationRule, strategyName string, svcPort proxy.ServicePortName, endpoints []string) {
	if _, exists := lb.strategyMap[svcPort]; !exists {
		switch strategyName {
		case RoundRobin:
			lb.strategyMap[svcPort] = NewRoundRobinStrategy()
		case Random:
			lb.strategyMap[svcPort] = NewRandomStrategy()
		case ConsistentHash:
			lb.strategyMap[svcPort] = NewConsistentHashStrategy(dr, endpoints)
		default:
			klog.Errorf("unsupported or empty load-balance strategy %s", strategyName)
			return
		}
	}
}

func isNamespacedNameEqual(dr *istioapi.DestinationRule, namespacedName *types.NamespacedName) bool {
	return dr.Namespace == namespacedName.Namespace && dr.Name == namespacedName.Name
}
