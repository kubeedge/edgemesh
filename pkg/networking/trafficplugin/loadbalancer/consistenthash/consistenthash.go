package consistenthash

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	hashring "github.com/buraksezer/consistent"
	"github.com/go-chassis/go-chassis/core/invocation"
	"github.com/go-chassis/go-chassis/core/registry"
	apiv1alpha3 "istio.io/api/networking/v1alpha3"
	istioapi "istio.io/client-go/pkg/apis/networking/v1alpha3"
	"k8s.io/klog/v2"

	"github.com/kubeedge/edgemesh/pkg/controller"
	"github.com/kubeedge/edgemesh/pkg/networking/util"
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
	s.instances = instances
	name, namespace := util.SplitServiceKey(serviceName)
	s.ring = fmt.Sprintf("%s.%s", namespace, name)

	// find destination rule bound to service
	dr, err := controller.GetDestinationRuleLister().DestinationRules(namespace).Get(name)
	if err != nil {
		klog.Errorf("failed to find destinationRule, reason: %v", err)
		return
	}

	// get key from request
	var hashKey string
	switch inv.Args.(type) {
	case *http.Request:
		hashKey, err = s.getKeyFromHTTP(inv, dr)
	case []byte: // tcp
		hashKey, err = s.getKeyFromTCP(inv, dr)
	default:
		err = fmt.Errorf("can't convert invocation.Args")
	}
	if err != nil {
		klog.Errorf("get key error: %v", err)
		return
	}
	klog.Infof("get key: %s", hashKey)
	s.key = hashKey
}

// Pick return instance
func (s *Strategy) Pick() (*registry.MicroServiceInstance, error) {
	hr, ok := GetHashRing(s.ring)
	if !ok {
		return nil, fmt.Errorf("can't find service instance hash ring %s", s.ring)
	}
	i := s.pick(hr)
	if i < 0 {
		klog.Errorf("can't find a service instance %d", i)
		return nil, fmt.Errorf("can't find a service instance")
	}
	return s.instances[i], nil
}

func (s *Strategy) pick(hr *hashring.Consistent) int {
	member := hr.LocateKey([]byte(s.key))
	if member == nil {
		klog.Errorf("can't find a home for given key %s", s.key)
		return -1
	}
	si, ok := member.(ServiceInstance)
	if !ok {
		klog.Errorf("can't convert to ServiceInstance")
		return -1
	}

	s.mtx.Lock()
	defer s.mtx.Unlock()

	for i, instance := range s.instances {
		if instance.ServiceID == si.String() {
			return i
		}
	}
	return -1
}

func (s *Strategy) getKeyFromHTTP(inv *invocation.Invocation, dr *istioapi.DestinationRule) (string, error) {
	req, ok := inv.Args.(*http.Request)
	if !ok {
		return "", fmt.Errorf("can't convert to http.Request")
	}
	hashKey, err := s.getKey(dr, "http", req, nil)
	if err != nil {
		return "", err
	}
	return hashKey, nil
}

func (s *Strategy) getKeyFromTCP(inv *invocation.Invocation, dr *istioapi.DestinationRule) (string, error) {
	// store tcp header fields
	req := make(map[string]string)
	data, ok := inv.Args.([]byte)
	if !ok {
		return "", fmt.Errorf("can't convert to []byte")
	}
	rd := bufio.NewReader(bytes.NewReader(data))
	for {
		line, err := rd.ReadString('\n')
		if err != nil || io.EOF == err {
			klog.Errorf("read tcp header fields error: %v", err)
			break
		}
		field := strings.Split(line, ": ")
		if len(field) == 2 {
			req[field[0]] = strings.Replace(field[1], "\r\n", "", -1)
		}
	}

	hashKey, err := s.getKey(dr, "tcp", nil, req)
	if err != nil {
		return "", err
	}
	return hashKey, nil
}

func (s *Strategy) getKey(dr *istioapi.DestinationRule, proto string,
	httpReq *http.Request, tcpReq map[string]string) (string, error) {
	var hashKey string
	switch lbPolicy := dr.Spec.TrafficPolicy.LoadBalancer.LbPolicy.(type) {
	case *apiv1alpha3.LoadBalancerSettings_Simple:
		return "", fmt.Errorf("hashkey can't get in loadBalancerSimple")
	case *apiv1alpha3.LoadBalancerSettings_ConsistentHash:
		switch consistentHashLb := lbPolicy.ConsistentHash.HashKey.(type) {
		case *apiv1alpha3.LoadBalancerSettings_ConsistentHashLB_HttpHeaderName:
			if "http" == proto {
				hashKey = httpReq.Header.Get(consistentHashLb.HttpHeaderName)
			} else { // tcp
				hashKey = tcpReq[consistentHashLb.HttpHeaderName]
			}
		case *apiv1alpha3.LoadBalancerSettings_ConsistentHashLB_HttpCookie:
			return "", fmt.Errorf("cookie as hashkey not support")
		case *apiv1alpha3.LoadBalancerSettings_ConsistentHashLB_UseSourceIp:
			if "http" == proto {
				hashKey = httpReq.Host
			} else { // tcp
				hashKey = tcpReq["Host"]
			}
		default:
			return "", fmt.Errorf("can't find ConsistentHash fields")
		}
	default:
		return "", fmt.Errorf("can't find LoadBalancerSettings")
	}
	return hashKey, nil
}
