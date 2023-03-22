package loadbalancer

import (
	"bufio"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"sync"

	"github.com/buraksezer/consistent"
	istioapi "istio.io/client-go/pkg/apis/networking/v1alpha3"
	"k8s.io/klog/v2"

	"github.com/kubeedge/edgemesh/pkg/apis/config/v1alpha1"
)

const (
	RoundRobin     = "ROUND_ROBIN"
	Random         = "RANDOM"
	ConsistentHash = "CONSISTENT_HASH"

	HttpHeader   = "HTTP_HEADER"
	UserSourceIP = "USER_SOURCE_IP"
)

type Policy interface {
	Name() string
	Update(oldDr, dr *istioapi.DestinationRule)
	Pick(endpoints []string, srcAddr net.Addr, tcpConn net.Conn, cliReq *http.Request) (string, *http.Request, error)
	Sync(endpoints []string)
	Release()
}

// RoundRobinPolicy is a default policy.
type RoundRobinPolicy struct {
}

func NewRoundRobinPolicy() *RoundRobinPolicy {
	return &RoundRobinPolicy{}
}

func (*RoundRobinPolicy) Name() string {
	return RoundRobin
}

func (*RoundRobinPolicy) Update(oldDr, dr *istioapi.DestinationRule) {}

func (*RoundRobinPolicy) Pick(endpoints []string, srcAddr net.Addr, netConn net.Conn, cliReq *http.Request) (string, *http.Request, error) {
	// RoundRobinPolicy is an empty implementation and we won't use it,
	// the outer round-robin policy will be used next.
	return "", cliReq, fmt.Errorf("call RoundRobinPolicy is forbidden")
}

func (*RoundRobinPolicy) Sync(endpoints []string) {}

func (*RoundRobinPolicy) Release() {}

type RandomPolicy struct {
	lock sync.Mutex
}

func NewRandomPolicy() *RandomPolicy {
	return &RandomPolicy{}
}

func (rd *RandomPolicy) Name() string {
	return Random
}

func (rd *RandomPolicy) Update(oldDr, dr *istioapi.DestinationRule) {}

func (rd *RandomPolicy) Pick(endpoints []string, srcAddr net.Addr, netConn net.Conn, cliReq *http.Request) (string, *http.Request, error) {
	rd.lock.Lock()
	k := rand.Int() % len(endpoints)
	rd.lock.Unlock()
	return endpoints[k], cliReq, nil
}

func (rd *RandomPolicy) Sync(endpoints []string) {}

func (rd *RandomPolicy) Release() {}

type ConsistentHashPolicy struct {
	Config   *v1alpha1.ConsistentHash
	lock     sync.Mutex
	hashRing *consistent.Consistent
	hashKey  HashKey
}

func NewConsistentHashPolicy(config *v1alpha1.ConsistentHash, dr *istioapi.DestinationRule, endpoints []string) *ConsistentHashPolicy {
	return &ConsistentHashPolicy{
		Config:   config,
		hashRing: newHashRing(config, endpoints),
		hashKey:  getConsistentHashKey(dr),
	}
}

func (ch *ConsistentHashPolicy) Name() string {
	return ConsistentHash
}

func (ch *ConsistentHashPolicy) Update(oldDr, dr *istioapi.DestinationRule) {
	ch.lock.Lock()
	ch.hashKey = getConsistentHashKey(dr)
	ch.lock.Unlock()
}

func (ch *ConsistentHashPolicy) Pick(endpoints []string, srcAddr net.Addr, netConn net.Conn, cliReq *http.Request) (endpoint string, req *http.Request, err error) {
	ch.lock.Lock()
	defer ch.lock.Unlock()

	req = cliReq
	var keyValue string
	switch ch.hashKey.Type {
	case HttpHeader:
		if req == nil {
			req, err = http.ReadRequest(bufio.NewReader(netConn))
			if err != nil {
				klog.Errorf("read http request err: %v", err)
				return "", nil, err
			}
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
	klog.Infof("Get key value: %s", keyValue)
	member := ch.hashRing.LocateKey([]byte(keyValue))
	if member == nil {
		errMsg := fmt.Errorf("can't find a endpoint by given key: %s", keyValue)
		klog.Errorf("%v", errMsg)
		return "", req, errMsg
	}
	return member.String(), req, nil
}

func (ch *ConsistentHashPolicy) Sync(endpoints []string) {
	ch.lock.Lock()
	if ch.hashRing == nil {
		ch.hashRing = newHashRing(ch.Config, endpoints)
	} else {
		updateHashRing(ch.hashRing, endpoints)
	}
	ch.lock.Unlock()
}

func (ch *ConsistentHashPolicy) Release() {
	ch.lock.Lock()
	clearHashRing(ch.hashRing)
	ch.lock.Unlock()
}
