package registry

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/go-chassis/go-chassis/core/registry"
	utiltags "github.com/go-chassis/go-chassis/pkg/util/tags"
	"github.com/kubeedge/edgemesh/agent/pkg/chassis/controller"
	"github.com/kubeedge/edgemesh/common/util"
	v1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
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
		return nil, fmt.Errorf("endpoint %s.%s is nil", namespace, name)
	}
	// get pods
	pods, err := controller.APIConn.GetPodLister().Pods(namespace).List(util.GetPodsSelector(svc))
	if err != nil {
		return nil, err
	}
	if len(pods) == 0 {
		return nil, fmt.Errorf("pod list is empty")
	}

	// get targetPort and Protocol from Service
	targetPort, proto := getPortAndProtocol(svc, svcPort)
	// port not found
	if targetPort == 0 {
		klog.Errorf("port %d not found in svc: %s.%s", svcPort, namespace, name)
		return nil, fmt.Errorf("port %d not found in svc: %s.%s", svcPort, namespace, name)
	}
	if proto == "http" {
		proto = "rest"
	}

	// gen
	var microServiceInstances instanceList
	var hostPort int32
	// all pods share the same host port, get from pods[0]
	if pods[0].Spec.HostNetwork {
		// host network
		hostPort = int32(targetPort)
	} else {
		// container network
		for _, container := range pods[0].Spec.Containers {
			for _, port := range container.Ports {
				if port.ContainerPort == int32(targetPort) {
					hostPort = port.HostPort
				}
			}
		}
	}
	// set targetPort from endpoints if hostPort == 0 still
	if hostPort == 0 {
		for _, a := range eps.Subsets {
			for _, port := range a.Ports {
				if port.Port != 0 {
					microServiceInstances = append(microServiceInstances, &registry.MicroServiceInstance{
						InstanceID:   fmt.Sprintf("%s.%s|%s.%d", namespace, name, a.Addresses[0].IP, port.Port),
						ServiceID:    fmt.Sprintf("%s#%s#%s", namespace, name, a.Addresses[0].IP),
						HostName:     "",
						EndpointsMap: map[string]string{proto: fmt.Sprintf("%s:%d", a.Addresses[0].IP, port.Port)},
					})
				}
			}
		}
	} else {
		// set Pod ip if hostPort != 0
		for _, p := range pods {
			if p.Status.Phase == v1.PodRunning {
				microServiceInstances = append(microServiceInstances, &registry.MicroServiceInstance{
					InstanceID:   fmt.Sprintf("%s.%s|%s.%d", namespace, name, p.Status.HostIP, hostPort),
					ServiceID:    fmt.Sprintf("%s#%s#%s", namespace, name, p.Status.HostIP),
					HostName:     "",
					EndpointsMap: map[string]string{proto: fmt.Sprintf("%s:%s:%d", p.Spec.NodeName, p.Status.HostIP, hostPort)},
				})
			}
		}
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

func getPortAndProtocol(svc *v1.Service, svcPort int) (targetPort int, protocol string) {
	for _, p := range svc.Spec.Ports {
		if p.Protocol == "TCP" && int(p.Port) == svcPort {
			protocol = strings.Split(p.Name, "-")[0]
			targetPort = p.TargetPort.IntValue()
			break
		}
	}
	return targetPort, protocol
}
