package loadbalancer

import (
	"github.com/buraksezer/consistent"
	"github.com/cespare/xxhash/v2"
	istiov1alpha3 "istio.io/api/networking/v1alpha3"
	istioapi "istio.io/client-go/pkg/apis/networking/v1alpha3"
	"k8s.io/klog/v2"

	"github.com/kubeedge/edgemesh/pkg/apis/config/v1alpha1"
)

type defaultHasher struct{}

func (h defaultHasher) Sum64(data []byte) uint64 {
	return xxhash.Sum64(data)
}

type ringItem string // "nodeName:podName:ip:port" style strings, nodeName and podName can be empty!!!

func (i ringItem) String() string {
	return string(i)
}

func newHashRing(config *v1alpha1.ConsistentHash, endpoints []string) *consistent.Consistent {
	members := []consistent.Member{}
	for i := 0; i < len(endpoints); i++ {
		member := ringItem(endpoints[i]) // alloc new string memory here.
		members = append(members, member)
	}
	// TODO read from container config
	cfg := consistent.Config{
		PartitionCount:    config.PartitionCount,
		ReplicationFactor: config.ReplicationFactor,
		Load:              config.Load,
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
	case *istiov1alpha3.LoadBalancerSettings_Simple:
		klog.Errorf("hash key can't get in LoadBalancerSettings_Simple")
		return HashKey{}
	case *istiov1alpha3.LoadBalancerSettings_ConsistentHash:
		if lbPolicy.ConsistentHash == nil {
			klog.Errorf("destination rule object %s .Spec.TrafficPolicy.LoadBalancer.LbPolicy.ConsistentHash is nil", getNamespaceName(dr))
			return HashKey{}
		}
		if lbPolicy.ConsistentHash.HashKey == nil {
			klog.Errorf("destination rule object %s .Spec.TrafficPolicy.LoadBalancer.LbPolicy.ConsistentHash.HashKey is nil", getNamespaceName(dr))
			return HashKey{}
		}
		switch consistentHashLb := lbPolicy.ConsistentHash.HashKey.(type) {
		case *istiov1alpha3.LoadBalancerSettings_ConsistentHashLB_HttpHeaderName:
			return HashKey{Type: HttpHeader, Key: consistentHashLb.HttpHeaderName}
		case *istiov1alpha3.LoadBalancerSettings_ConsistentHashLB_HttpCookie:
			klog.Errorf("http cookie is not supported as a hash key")
			return HashKey{}
		case *istiov1alpha3.LoadBalancerSettings_ConsistentHashLB_UseSourceIp:
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
