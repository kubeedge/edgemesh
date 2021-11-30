package controller

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	"github.com/kubeedge/edgemesh/agent/pkg/chassis/protocol"
	"github.com/kubeedge/edgemesh/common/informers"
)

const (
	None                 = "None"
	defaultNetworkPrefix = "9.251."
)

var (
	APIConn *ProxyController
	once    sync.Once
)

type ProxyController struct {
	FakeIPIndex      uint16
	svcInformer      cache.SharedIndexInformer
	svcEventHandlers map[string]cache.ResourceEventHandlerFuncs // key: service event handler name

	sync.RWMutex
	svcPortsByIP map[string]string // key: clusterIP, value: SvcPorts
	ipBySvc      map[string]string // key: svcNamespace.svcName, value: clusterIP
}

func Init(ifm *informers.Manager) {
	once.Do(func() {
		APIConn = &ProxyController{
			FakeIPIndex:      uint16(1), // avoid 0.0
			svcInformer:      ifm.GetKubeFactory().Core().V1().Services().Informer(),
			svcEventHandlers: make(map[string]cache.ResourceEventHandlerFuncs),
			svcPortsByIP:     make(map[string]string),
			ipBySvc:          make(map[string]string),
		}
		ifm.RegisterInformer(APIConn.svcInformer)
		ifm.RegisterSyncedFunc(APIConn.onCacheSynced)

		// set api-connection-service event handler funcs
		APIConn.SetServiceEventHandlers("api-connection-service", cache.ResourceEventHandlerFuncs{
			AddFunc: APIConn.svcAdd, UpdateFunc: APIConn.svcUpdate, DeleteFunc: APIConn.svcDelete})
	})
}

func (c *ProxyController) onCacheSynced() {
	for name, funcs := range c.svcEventHandlers {
		klog.V(4).Infof("enable service event handler funcs: %s", name)
		c.svcInformer.AddEventHandler(funcs)
	}
}

func (c *ProxyController) SetServiceEventHandlers(name string, handlerFuncs cache.ResourceEventHandlerFuncs) {
	c.Lock()
	if _, exist := c.svcEventHandlers[name]; exist {
		klog.Warningf("service event handler %s already exists, it will be overwritten!", name)
	}
	c.svcEventHandlers[name] = handlerFuncs
	c.Unlock()
}

func (c *ProxyController) setFakeIPIndex(newIndex uint16) {
	c.FakeIPIndex = newIndex
}

func (c *ProxyController) getFakeIPIndex() uint16 {
	return c.FakeIPIndex
}

func getSvcPorts(svc *v1.Service) string {
	svcPorts := ""
	svcName := svc.Namespace + "." + svc.Name
	for _, p := range svc.Spec.Ports {
		var protocolName string
		pro := strings.Split(p.Name, "-")[0]
		for _, p := range protocol.RegisterProtocols {
			if p == pro {
				protocolName = pro
				break
			}
		}
		if protocolName == "" {
			protocolName = strings.ToLower(string(p.Protocol))
		}
		sub := fmt.Sprintf("%s,%d,%d|", protocolName, p.Port, p.TargetPort.IntVal)
		svcPorts = svcPorts + sub
	}
	svcPorts += svcName
	return svcPorts
}

func (c *ProxyController) svcAdd(obj interface{}) {
	svc, ok := obj.(*v1.Service)
	if !ok {
		klog.Errorf("invalid type %v", obj)
		return
	}
	svcPorts := getSvcPorts(svc)
	svcName := svc.Namespace + "." + svc.Name
	ip := svc.Spec.ClusterIP
	if ip == "" || ip == None {
		ip = GetFakeIP()
		klog.Warningf("[EdgeMesh] use fakeIP for service %s, fakeIP:%s", svcName, ip)
	}
	c.addOrUpdateService(svcName, ip, svcPorts)
}

func (c *ProxyController) svcUpdate(oldObj, newObj interface{}) {
	svc, ok := newObj.(*v1.Service)
	if !ok {
		klog.Errorf("invalid type %v", newObj)
		return
	}
	svcPorts := getSvcPorts(svc)
	svcName := svc.Namespace + "." + svc.Name
	ip := svc.Spec.ClusterIP

	clusterIP := ip
	old, ok := c.ipBySvc[svcName]
	if ok {
		clusterIP = old
	} else {
		if ip == "" || ip == None {
			clusterIP = GetFakeIP()
			klog.Warningf("[EdgeMesh] use fakeIP for service %s, fakeIP:%s", svcName, clusterIP)
		}
	}
	c.addOrUpdateService(svcName, clusterIP, svcPorts)
}

func (c *ProxyController) svcDelete(obj interface{}) {
	svc, ok := obj.(*v1.Service)
	if !ok {
		klog.Errorf("invalid type %v", obj)
		return
	}
	svcName := svc.Namespace + "." + svc.Name
	ip := svc.Spec.ClusterIP
	c.deleteService(svcName, ip)
}

// AddOrUpdateService add or updates a service
func (c *ProxyController) addOrUpdateService(svcName, ip, svcPorts string) {
	c.Lock()
	c.ipBySvc[svcName] = ip
	c.svcPortsByIP[ip] = svcPorts
	c.Unlock()
}

// DeleteService deletes a service
func (c *ProxyController) deleteService(svcName, ip string) {
	c.Lock()
	delete(c.ipBySvc, svcName)
	delete(c.svcPortsByIP, ip)
	c.Unlock()
}

// GetSvcIP returns the ip by given service name
func (c *ProxyController) GetSvcIP(svcName string) string {
	c.RLock()
	ip := c.ipBySvc[svcName]
	c.RUnlock()
	return ip
}

// GetSvcPorts is a thread-safe operation to get from map
func (c *ProxyController) GetSvcPorts(ip string) string {
	c.RLock()
	svcPorts := c.svcPortsByIP[ip]
	c.RUnlock()
	return svcPorts
}

func GetFakeIP() (ip string) {
	for {
		index := APIConn.getFakeIPIndex() + 1
		APIConn.setFakeIPIndex(index)

		ip = defaultNetworkPrefix + getSubNet(index)

		_, ok := APIConn.svcPortsByIP[ip]
		if !ok {
			break
		}

	}
	return
}

// getSubNet converts uint16 to "uint8.uint8"
func getSubNet(subNet uint16) string {
	arg1 := uint64(subNet & 0x00ff)
	arg2 := uint64((subNet & 0xff00) >> 8)
	return strconv.FormatUint(arg2, 10) + "." + strconv.FormatUint(arg1, 10)
}
