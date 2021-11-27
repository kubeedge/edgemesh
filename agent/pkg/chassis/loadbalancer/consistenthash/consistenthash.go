package consistenthash

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/buraksezer/consistent"
	"github.com/go-chassis/go-chassis/core/invocation"
	"github.com/go-chassis/go-chassis/core/registry"
	apiv1alpha3 "istio.io/api/networking/v1alpha3"
	istioapi "istio.io/client-go/pkg/apis/networking/v1alpha3"
	"k8s.io/klog/v2"

	"github.com/kubeedge/edgemesh/agent/pkg/chassis/controller"
	"github.com/kubeedge/edgemesh/agent/pkg/chassis/loadbalancer/consistenthash/hashring"
	"github.com/kubeedge/edgemesh/common/util"
)

// StrategyConsistentHash load balance strategy
const StrategyConsistentHash = "ConsistentHash"

// Strategy is an extension of the go-chassis loadbalancer
type Strategy struct {
	instances []*registry.MicroServiceInstance
	mtx       sync.Mutex
	key, ring string
}

// ReceiveData receive data.
func (s *Strategy) ReceiveData(inv *invocation.Invocation,
	instances []*registry.MicroServiceInstance, serviceName string) {
	s.mtx.Lock()
	s.instances = instances
	s.mtx.Unlock()
	name, namespace := util.SplitServiceKey(serviceName)
	s.ring = fmt.Sprintf("%s.%s", namespace, name)

	// find destination rule bound to service
	dr, err := controller.APIConn.GetDrLister().DestinationRules(namespace).Get(name)
	if err != nil {
		klog.Errorf("failed to find destinationRule, reason: %v", err)
		return
	}

	// get key from inv
	key, err := s.getKey(inv, dr)
	if err != nil {
		klog.Errorf("get key error: %v", err)
	} else {
		klog.Infof("get key: %s", key)
	}
	s.key = key
}

// Pick return instance
func (s *Strategy) Pick() (*registry.MicroServiceInstance, error) {
	hr, ok := hashring.GetHashRing(s.ring)
	if !ok {
		return nil, fmt.Errorf("can't find service instance hash ring %s", s.ring)
	}

	return s.pick(hr)
}

func (s *Strategy) getKey(inv *invocation.Invocation, dr *istioapi.DestinationRule) (string, error) {
	data, ok := inv.Args.([]byte)
	if !ok {
		return "", fmt.Errorf("can't convert to []byte")
	}
	req, err := http.ReadRequest(bufio.NewReader(bytes.NewReader(data)))
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("read http request err: %w", err)
	}

	var hashKey string
	switch lbPolicy := dr.Spec.TrafficPolicy.LoadBalancer.LbPolicy.(type) {
	case *apiv1alpha3.LoadBalancerSettings_Simple:
		return "", fmt.Errorf("hashkey can't get in loadBalancerSimple")
	case *apiv1alpha3.LoadBalancerSettings_ConsistentHash:
		switch consistentHashLb := lbPolicy.ConsistentHash.HashKey.(type) {
		case *apiv1alpha3.LoadBalancerSettings_ConsistentHashLB_HttpHeaderName:
			hashKey = req.Header.Get(consistentHashLb.HttpHeaderName)
		case *apiv1alpha3.LoadBalancerSettings_ConsistentHashLB_HttpCookie:
			return "", fmt.Errorf("cookie as hashkey not support")
		case *apiv1alpha3.LoadBalancerSettings_ConsistentHashLB_UseSourceIp:
			hashKey = req.Host
		default:
			return "", fmt.Errorf("can't find ConsistentHash fields")
		}
	default:
		return "", fmt.Errorf("can't find LoadBalancerSettings")
	}
	return hashKey, nil
}

func (s *Strategy) pick(hr *consistent.Consistent) (*registry.MicroServiceInstance, error) {
	member := hr.LocateKey([]byte(s.key))
	if member == nil {
		errMsg := fmt.Errorf("can't find a home for given key %s", s.key)
		klog.Errorf("%v", errMsg)
		return nil, errMsg
	}
	si, ok := member.(hashring.ServiceInstance)
	if !ok {
		errMsg := fmt.Errorf("%T can't convert to ServiceInstance", member)
		klog.Errorf("%v", errMsg)
		return nil, errMsg
	}

	s.mtx.Lock()
	defer s.mtx.Unlock()

	for i, instance := range s.instances {
		if instance.ServiceID == si.String() {
			return s.instances[i], nil
		}
	}
	return nil, fmt.Errorf("service instance not exist")
}
