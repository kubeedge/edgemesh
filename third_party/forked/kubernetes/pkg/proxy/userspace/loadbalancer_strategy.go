package userspace

import (
	"bufio"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"sync"

	"github.com/buraksezer/consistent"
	"github.com/cespare/xxhash/v2"
	apiv1alpha3 "istio.io/api/networking/v1alpha3"
	istioapi "istio.io/client-go/pkg/apis/networking/v1alpha3"
	"k8s.io/klog/v2"
)

const (
	RoundRobin     = "ROUND_ROBIN"
	Random         = "RANDOM"
	ConsistentHash = "CONSISTENT_HASH"

	EmptyNodeName = "EMPTY_NODE_NAME"
	EmptyPodName  = "EMPTY_POD_NAME"
)

type LoadBalancerStrategy interface {
	Name() string
	Update(oldDr, dr *istioapi.DestinationRule)
	Pick(endpoints []string, srcAddr net.Addr, tcpConn net.Conn) (string, *http.Request, error)
	Sync(endpoints []string)
	Release()
}

func getNamespaceName(dr *istioapi.DestinationRule) string {
	return fmt.Sprintf("%s/%s", dr.Namespace, dr.Name)
}

// getStrategyName gets strategy name from a DestinationRule object.
func getStrategyName(dr *istioapi.DestinationRule) string {
	if dr.Spec.TrafficPolicy == nil {
		klog.Errorf("destination rule object %s .Spec.TrafficPolicy is nil", getNamespaceName(dr))
		return ""
	}
	if dr.Spec.TrafficPolicy.LoadBalancer == nil {
		klog.Errorf("destination rule object %s .Spec.TrafficPolicy.LoadBalancer is nil", getNamespaceName(dr))
		return ""
	}
	if dr.Spec.TrafficPolicy.LoadBalancer.LbPolicy == nil {
		klog.Errorf("destination rule object %s .Spec.TrafficPolicy.LoadBalancer.LbPolicy is nil", getNamespaceName(dr))
		return ""
	}
	switch lbPolicy := dr.Spec.TrafficPolicy.LoadBalancer.LbPolicy.(type) {
	case *apiv1alpha3.LoadBalancerSettings_Simple:
		return lbPolicy.Simple.String()
	case *apiv1alpha3.LoadBalancerSettings_ConsistentHash:
		return ConsistentHash
	default:
		klog.Errorf("unsupported load balancer policy %v", lbPolicy)
		return ""
	}
}

// RoundRobinStrategy is a default strategy.
type RoundRobinStrategy struct {
}

func NewRoundRobinStrategy() *RoundRobinStrategy {
	return &RoundRobinStrategy{}
}

func (*RoundRobinStrategy) Name() string {
	return RoundRobin
}

func (*RoundRobinStrategy) Update(oldDr, dr *istioapi.DestinationRule) {}

func (*RoundRobinStrategy) Pick(endpoints []string, srcAddr net.Addr, netConn net.Conn) (string, *http.Request, error) {
	// RoundRobinStrategy is an empty implementation and we won't use it,
	// the outer round-robin strategy will be used next.
	return "", nil, fmt.Errorf("call RoundRobinStrategy is forbidden")
}

func (*RoundRobinStrategy) Sync(endpoints []string) {}

func (*RoundRobinStrategy) Release() {}

type RandomStrategy struct {
	lock sync.Mutex
}

func NewRandomStrategy() *RandomStrategy {
	return &RandomStrategy{}
}

func (rd *RandomStrategy) Name() string {
	return Random
}

func (rd *RandomStrategy) Update(oldDr, dr *istioapi.DestinationRule) {}

func (rd *RandomStrategy) Pick(endpoints []string, srcAddr net.Addr, netConn net.Conn) (string, *http.Request, error) {
	rd.lock.Lock()
	k := rand.Int() % len(endpoints)
	rd.lock.Unlock()
	return endpoints[k], nil, nil
}

func (rd *RandomStrategy) Sync(endpoints []string) {}

func (rd *RandomStrategy) Release() {}

type defaultHasher struct{}

func (h defaultHasher) Sum64(data []byte) uint64 {
	return xxhash.Sum64(data)
}

type ringItem string // "nodeName:podName:ip:port" style strings, nodeName and podName can be empty!!!

func (i ringItem) String() string {
	return string(i)
}

func newHashRing(endpoints []string) *consistent.Consistent {
	members := []consistent.Member{}
	for i := 0; i < len(endpoints); i++ {
		member := ringItem(fmt.Sprintf("%s", endpoints[i])) // alloc new string memory here.
		members = append(members, member)
	}
	// TODO read from container config
	cfg := consistent.Config{
		PartitionCount:    100,
		ReplicationFactor: 10,
		Load:              1.25,
		Hasher:            defaultHasher{},
	}
	// If len(members) is equal to 0, consistent.New(members, cfg) will cause
	// divide zero panic, see issue: https://github.com/buraksezer/consistent/issues/19
	if len(members) == 0 {
		return consistent.New(nil, cfg)
	}
	return consistent.New(members, cfg)
}

func updateHashRing(hr *consistent.Consistent, endpoints []string) {
	if hr == nil {
		return
	}
	oldEndpoints := []string{}
	for _, member := range hr.GetMembers() {
		oldEndpoints = append(oldEndpoints, member.String())
	}
	klog.Infof("oldEndpoints: %v", oldEndpoints)
	klog.Infof("newEndpoints: %v", endpoints)
	addedItems, deletedItems := sliceCompare(oldEndpoints, endpoints)
	for _, item := range addedItems {
		klog.Infof("add item %s to hash ring", item)
		hr.Add(ringItem(item))
	}
	for _, item := range deletedItems {
		klog.Infof("delete item %s from hash ring", item)
		hr.Remove(item)
	}
}

// sliceCompare finds the difference between two string slice.
func sliceCompare(src []string, dest []string) ([]string, []string) {
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

func clearHashRing(hr *consistent.Consistent) {
	if hr == nil {
		return
	}
	for _, item := range hr.GetMembers() {
		hr.Remove(item.String())
	}
	// Reference count is 0, waiting for GC to clean up?
	hr = nil
}

const (
	// supported consistent hash keys
	HttpHeader   = "HTTP_HEADER"
	UserSourceIP = "USER_SOURCE_IP"
)

type HashKey struct {
	Type string
	Key  string
}

// getConsistentHashKey get consistent hash key from destination rule object.
func getConsistentHashKey(dr *istioapi.DestinationRule) HashKey {
	if dr.Spec.TrafficPolicy == nil {
		klog.Errorf("destination rule object %s .Spec.TrafficPolicy is nil", getNamespaceName(dr))
		return HashKey{}
	}
	if dr.Spec.TrafficPolicy.LoadBalancer == nil {
		klog.Errorf("destination rule object %s .Spec.TrafficPolicy.LoadBalancer is nil", getNamespaceName(dr))
		return HashKey{}
	}
	if dr.Spec.TrafficPolicy.LoadBalancer.LbPolicy == nil {
		klog.Errorf("destination rule object %s .Spec.TrafficPolicy.LoadBalancer.LbPolicy is nil", getNamespaceName(dr))
		return HashKey{}
	}

	switch lbPolicy := dr.Spec.TrafficPolicy.LoadBalancer.LbPolicy.(type) {
	case *apiv1alpha3.LoadBalancerSettings_Simple:
		klog.Errorf("hash key can't get in LoadBalancerSettings_Simple")
		return HashKey{}
	case *apiv1alpha3.LoadBalancerSettings_ConsistentHash:
		if lbPolicy.ConsistentHash == nil {
			klog.Errorf("destination rule object %s .Spec.TrafficPolicy.LoadBalancer.LbPolicy.ConsistentHash is nil", getNamespaceName(dr))
			return HashKey{}
		}
		if lbPolicy.ConsistentHash.HashKey == nil {
			klog.Errorf("destination rule object %s .Spec.TrafficPolicy.LoadBalancer.LbPolicy.ConsistentHash.HashKey is nil", getNamespaceName(dr))
			return HashKey{}
		}
		switch consistentHashLb := lbPolicy.ConsistentHash.HashKey.(type) {
		case *apiv1alpha3.LoadBalancerSettings_ConsistentHashLB_HttpHeaderName:
			return HashKey{Type: HttpHeader, Key: consistentHashLb.HttpHeaderName}
		case *apiv1alpha3.LoadBalancerSettings_ConsistentHashLB_HttpCookie:
			klog.Errorf("http cookie is not supported as a hash key")
			return HashKey{}
		case *apiv1alpha3.LoadBalancerSettings_ConsistentHashLB_UseSourceIp:
			return HashKey{Type: UserSourceIP, Key: ""}
		default:
			klog.Errorf("%s unsupported ConsistentHash fields", getNamespaceName(dr))
			return HashKey{}
		}
	default:
		klog.Errorf("%s unsupported LoadBalancer fields", getNamespaceName(dr))
		return HashKey{}
	}
}

type ConsistentHashStrategy struct {
	lock     sync.Mutex
	hashRing *consistent.Consistent
	hashKey  HashKey
}

func NewConsistentHashStrategy(dr *istioapi.DestinationRule, endpoints []string) *ConsistentHashStrategy {
	return &ConsistentHashStrategy{
		hashRing: newHashRing(endpoints),
		hashKey:  getConsistentHashKey(dr),
	}
}

func (ch *ConsistentHashStrategy) Name() string {
	return ConsistentHash
}

func (ch *ConsistentHashStrategy) Update(oldDr, dr *istioapi.DestinationRule) {
	ch.lock.Lock()
	ch.hashKey = getConsistentHashKey(dr)
	ch.lock.Unlock()
}

func (ch *ConsistentHashStrategy) Pick(endpoints []string, srcAddr net.Addr, netConn net.Conn) (endpoint string, req *http.Request, err error) {
	ch.lock.Lock()
	defer ch.lock.Unlock()

	var keyValue string
	switch ch.hashKey.Type {
	case HttpHeader:
		req, err = http.ReadRequest(bufio.NewReader(netConn))
		if err != nil {
			klog.Errorf("read http request err: %v", err)
			return "", nil, err
		}
		keyValue = req.Header.Get(ch.hashKey.Key)
	case UserSourceIP:
		if srcAddr == nil && netConn != nil {
			srcAddr = netConn.RemoteAddr()
		}
		keyValue = srcAddr.String()
	default:
		klog.Errorf("Failed to get hash key value")
		keyValue = ""
	}
	klog.Infof("get key value: %s", keyValue)
	member := ch.hashRing.LocateKey([]byte(keyValue))
	if member == nil {
		errMsg := fmt.Errorf("can't find a endpoint by given key: %s", keyValue)
		klog.Errorf("%v", errMsg)
		return "", req, errMsg
	}
	return member.String(), req, nil
}

func (ch *ConsistentHashStrategy) Sync(endpoints []string) {
	ch.lock.Lock()
	if ch.hashRing == nil {
		ch.hashRing = newHashRing(endpoints)
	} else {
		updateHashRing(ch.hashRing, endpoints)
	}
	ch.lock.Unlock()
}

func (ch *ConsistentHashStrategy) Release() {
	ch.lock.Lock()
	clearHashRing(ch.hashRing)
	ch.lock.Unlock()
}
