package registry

import (
	"fmt"
	"sort"
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

type instanceList []*registry.MicroServiceInstance

func (I instanceList) Len() int {
	return len(I)
}

func (I instanceList) Less(i, j int) bool {
	return strings.Compare(I[i].InstanceID, I[j].InstanceID) < 0
}

func (I instanceList) Swap(i, j int) {
	I[i], I[j] = I[j], I[i]
}

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
	var microServiceInstances instanceList

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
	for _, subset := range eps.Subsets {
		for _, addr := range subset.Addresses {
			for _, port := range subset.Ports {
				if addr.NodeName == nil || port.Port != int32(targetPort) {
					// Each backend(Address) must have a nodeName, so we do not support custom Endpoints now.
					// This means that external services cannot be used.
					continue
				}

				microServiceInstances = append(microServiceInstances, &registry.MicroServiceInstance{
					InstanceID:   fmt.Sprintf("%s.%s|%s.%d", namespace, name, addr.IP, targetPort),
					ServiceID:    fmt.Sprintf("%s#%s#%s", namespace, name, addr.TargetRef.Name),
					HostName:     "",
					EndpointsMap: map[string]string{proto: fmt.Sprintf("%s:%s:%d", *addr.NodeName, addr.IP, targetPort)},
				})
			}
		}
	}

	// no instances found
	if len(microServiceInstances) == 0 {
		return nil, fmt.Errorf("service %s.%s has no instances", namespace, name)
	}

	// Why do we need to sort microServiceInstances?
	// That's because the pod list obtained by the PodLister is out of order.
	sort.Sort(microServiceInstances)

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
