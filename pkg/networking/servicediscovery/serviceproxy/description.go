package serviceproxy

import (
	"sync"
)

var svcDesc *ServiceDescription

type ServiceDescription struct {
	sync.RWMutex
	SvcPortsByIP map[string]string // key: clusterIP, value: SvcPorts
	IPBySvc      map[string]string // key: svcName.svcNamespace, value: clusterIP
}

func newServiceDescription() *ServiceDescription {
	return &ServiceDescription{
		SvcPortsByIP: make(map[string]string),
		IPBySvc:      make(map[string]string),
	}
}

// set is a thread-safe operation to add to map
func (sd *ServiceDescription) set(svcName, ip, svcPorts string) {
	sd.Lock()
	defer sd.Unlock()
	sd.IPBySvc[svcName] = ip
	sd.SvcPortsByIP[ip] = svcPorts
}

// del is a thread-safe operation to del from map
func (sd *ServiceDescription) del(svcName, ip string) {
	sd.Lock()
	defer sd.Unlock()
	delete(sd.IPBySvc, svcName)
	delete(sd.SvcPortsByIP, ip)
}

// getIP is a thread-safe operation to get from map
func (sd *ServiceDescription) getIP(svcName string) string {
	sd.RLock()
	defer sd.RUnlock()
	ip := sd.IPBySvc[svcName]
	return ip
}

// getSvcPorts is a thread-safe operation to get from map
func (sd *ServiceDescription) getSvcPorts(ip string) string {
	sd.RLock()
	defer sd.RUnlock()
	svcPorts := sd.SvcPortsByIP[ip]
	return svcPorts
}

// GetServiceClusterIP returns the proxier IP by given name
func GetServiceClusterIP(svcName string) string {
	ip := svcDesc.getIP(svcName)
	return ip
}

// AddOrUpdateService add or updates a service
func AddOrUpdateService(svcName, clusterIP, svcPorts string) {
	svcDesc.set(svcName, clusterIP, svcPorts)
}

// DeleteService deletes a service
func DeleteService(svcName, clusterIP string) {
	svcDesc.del(svcName, clusterIP)
}
