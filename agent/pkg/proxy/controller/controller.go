package controller

import (
	"fmt"
	"strings"
	"sync"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	"github.com/kubeedge/edgemesh/common/informers"
)

var (
	APIConn *ProxyController
	once    sync.Once
)

type ProxyController struct {
	svcInformer cache.SharedIndexInformer

	sync.RWMutex
	svcPortsByIP map[string]string // key: clusterIP, value: SvcPorts
	ipBySvc      map[string]string // key: svcName.svcNamespace, value: clusterIP
}

func Init(ifm *informers.Manager) {
	once.Do(func() {
		APIConn = &ProxyController{
			svcInformer:  ifm.GetKubeFactory().Core().V1().Services().Informer(),
			svcPortsByIP: make(map[string]string),
			ipBySvc:      make(map[string]string),
		}
		ifm.RegisterInformer(APIConn.svcInformer)
		ifm.RegisterSyncedFunc(APIConn.onCacheSynced)
	})
}

func (c *ProxyController) onCacheSynced() {
	// set informers event handler
	c.svcInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.svcAdd, UpdateFunc: c.svcUpdate, DeleteFunc: c.svcDelete})
}

func getSvcPorts(svc *v1.Service) string {
	svcPorts := ""
	svcName := svc.Namespace + "." + svc.Name
	for _, p := range svc.Spec.Ports {
		pro := strings.Split(p.Name, "-")
		sub := fmt.Sprintf("%s,%d,%d|", pro[0], p.Port, p.TargetPort.IntVal)
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
	if ip == "" || ip == "None" {
		return
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
	if ip == "" || ip == "None" {
		return
	}
	c.addOrUpdateService(svcName, ip, svcPorts)
}

func (c *ProxyController) svcDelete(obj interface{}) {
	svc, ok := obj.(*v1.Service)
	if !ok {
		klog.Errorf("invalid type %v", obj)
		return
	}
	svcName := svc.Namespace + "." + svc.Name
	ip := svc.Spec.ClusterIP
	if ip == "" || ip == "None" {
		return
	}
	c.deleteService(svcName, ip)
}

// AddOrUpdateService add or updates a service
func (c *ProxyController) addOrUpdateService(svcName, ip, svcPorts string) {
	c.Lock()
	defer c.Unlock()
	c.ipBySvc[svcName] = ip
	c.svcPortsByIP[ip] = svcPorts
}

// DeleteService deletes a service
func (c *ProxyController) deleteService(svcName, ip string) {
	c.Lock()
	defer c.Unlock()
	delete(c.ipBySvc, svcName)
	delete(c.svcPortsByIP, ip)
}

// GetSvcIP returns the ip by given service name
func (c *ProxyController) GetSvcIP(svcName string) string {
	c.RLock()
	defer c.RUnlock()
	ip := c.ipBySvc[svcName]
	return ip
}

// GetSvcPorts is a thread-safe operation to get from map
func (c *ProxyController) GetSvcPorts(ip string) string {
	c.RLock()
	defer c.RUnlock()
	svcPorts := c.svcPortsByIP[ip]
	return svcPorts
}
