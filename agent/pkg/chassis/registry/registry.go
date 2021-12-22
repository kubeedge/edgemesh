package registry

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/go-chassis/go-chassis/core/registry"
	utiltags "github.com/go-chassis/go-chassis/pkg/util/tags"
	v1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"

	"github.com/kubeedge/edgemesh/agent/pkg/chassis/controller"
	"github.com/kubeedge/edgemesh/agent/pkg/chassis/protocol"
	"github.com/kubeedge/edgemesh/common/util"
)

const (
	// EdgeRegistry constants string
	EdgeRegistry = "edge"
)

// init initialize the service discovery of edge meta registry
func init() {
	registry.InstallServiceDiscovery(EdgeRegistry, NewEdgeServiceDiscovery)
}

// EdgeServiceDiscovery to represent the object of service center to call the APIs of service center
type EdgeServiceDiscovery struct {
	Name string
}

func NewEdgeServiceDiscovery(options registry.Options) registry.ServiceDiscovery {
	return &EdgeServiceDiscovery{
		Name: EdgeRegistry,
	}
}

// GetAllMicroServices Get all MicroService information.
func (esd *EdgeServiceDiscovery) GetAllMicroServices() ([]*registry.MicroService, error) {
	return nil, nil
}

// FindMicroServiceInstances find micro-service instances (subnets)
func (esd *EdgeServiceDiscovery) FindMicroServiceInstances(consumerID, microServiceName string, tags utiltags.Tags) ([]*registry.MicroServiceInstance, error) {
	var microServiceInstances []*registry.MicroServiceInstance

	// parse microServiceName
	name, namespace, svcPort, err := parseServiceURL(microServiceName)
	if err != nil {
		return nil, err
	}

	// get service
	svc, err := controller.APIConn.GetSvcLister().Services(namespace).Get(name)
	if err != nil {
		return nil, err
	}
	if svc == nil {
		return nil, fmt.Errorf("service %s.%s is nil", namespace, name)
	}

	// get endpoints
	eps, err := controller.APIConn.GetEpLister().Endpoints(namespace).Get(name)
	if err != nil {
		return nil, err
	}
	if eps == nil {
		return nil, fmt.Errorf("endpoints %s.%s is nil", namespace, name)
	}

	// get targetPort and protocol from Service
	targetPort, proto := getPortAndProtocol(svc, svcPort)
	// targetPort not found
	if targetPort == 0 {
		return nil, fmt.Errorf("targetPort %d not found in service: %s.%s", svcPort, namespace, name)
	}

	// transport http data through tcp transparently
	if proto == "http" {
		proto = "tcp"
	}

	// gen MicroServiceInstances
	var nodeName string
	var instanceName string
	for _, subset := range eps.Subsets {
		for _, addr := range subset.Addresses {
			if addr.NodeName != nil {
				nodeName = *addr.NodeName
			} else {
				nodeName = ""
			}
			if addr.TargetRef != nil {
				instanceName = addr.TargetRef.Name
			} else {
				instanceName = addr.IP
			}
			microServiceInstances = append(microServiceInstances, &registry.MicroServiceInstance{
				InstanceID:   fmt.Sprintf("%s.%s|%s:%s:%d", namespace, name, nodeName, addr.IP, targetPort),
				ServiceID:    fmt.Sprintf("%s#%s#%s", namespace, name, instanceName),
				EndpointsMap: map[string]string{proto: fmt.Sprintf("%s:%s:%d", nodeName, addr.IP, targetPort)},
			})
		}
	}

	return microServiceInstances, nil
}

// GetMicroServiceID get microServiceID
func (esd *EdgeServiceDiscovery) GetMicroServiceID(appID, microServiceName, version, env string) (string, error) {
	return "", nil
}

// GetMicroServiceInstances return instances
func (esd *EdgeServiceDiscovery) GetMicroServiceInstances(consumerID, providerID string) ([]*registry.MicroServiceInstance, error) {
	return nil, nil
}

// GetMicroService return service
func (esd *EdgeServiceDiscovery) GetMicroService(microServiceID string) (*registry.MicroService, error) {
	return nil, nil
}

// AutoSync updating the cache manager
func (esd *EdgeServiceDiscovery) AutoSync() {}

// Close close all websocket connection
func (esd *EdgeServiceDiscovery) Close() error { return nil }

// parseServiceURL parses serviceURL to ${service_name}.${namespace}.svc.${cluster}:${port}, keeps with k8s service
func parseServiceURL(serviceURL string) (string, string, int, error) {
	var port int
	var err error
	serviceURLSplit := strings.Split(serviceURL, ":")
	if len(serviceURLSplit) == 1 {
		// default
		port = 80
	} else if len(serviceURLSplit) == 2 {
		port, err = strconv.Atoi(serviceURLSplit[1])
		if err != nil {
			klog.Errorf("service url %s invalid", serviceURL)
			return "", "", 0, err
		}
	} else {
		klog.Errorf("service url %s invalid", serviceURL)
		err = fmt.Errorf("service url %s invalid", serviceURL)
		return "", "", 0, err
	}
	name, namespace := util.SplitServiceKey(serviceURLSplit[0])
	return name, namespace, port, nil
}

// getPortAndProtocol get targetPort and protocol, targetPort is equal to containerPort
func getPortAndProtocol(svc *v1.Service, svcPort int) (targetPort int, protocolName string) {
	for _, p := range svc.Spec.Ports {
		if p.Protocol == "TCP" && int(p.Port) == svcPort {
			protocolName = "tcp"
			pro := strings.Split(p.Name, "-")[0]
			for _, p := range protocol.RegisterProtocols {
				if p == pro {
					protocolName = pro
					break
				}
			}
			targetPort = p.TargetPort.IntValue()
			break
		}
	}
	return targetPort, protocolName
}
