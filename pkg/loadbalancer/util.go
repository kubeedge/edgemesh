package loadbalancer

import (
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"

	istiov1alpha3 "istio.io/api/networking/v1alpha3"
	istioapi "istio.io/client-go/pkg/apis/networking/v1alpha3"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/proxy"
	stringslices "k8s.io/utils/strings/slices"

	"github.com/kubeedge/edgemesh/pkg/apis/config/defaults"
)

const (
	nodeIndex = iota
	podIndex
	ipIndex
	portIndex
	endpointLen
)

// parseEndpoint parse an endpoint like "nodeName:podName:ip:port" style strings
func parseEndpoint(endpoint string) (string, string, string, string, bool) {
	info := strings.Split(endpoint, ":")
	if len(info) != endpointLen {
		return "", "", "", "", false
	}
	// TODO check IP and port
	return info[nodeIndex], info[podIndex], info[ipIndex], info[portIndex], true
}

// isValidEndpoint checks that the given host / port pair are valid endpoint
func isValidEndpoint(host string, port int) bool {
	return host != "" && port > 0
}

// buildPortsToEndpointsMap builds a map of portname -> all nodeName:podName:ip:port
// for that portname. Explode Endpoints.Subsets[*] into this structure.
func buildPortsToEndpointsMap(endpoints *v1.Endpoints) map[string][]string {
	portsToEndpoints := map[string][]string{}
	for i := range endpoints.Subsets {
		ss := &endpoints.Subsets[i]
		for i := range ss.Ports {
			port := &ss.Ports[i]
			for i := range ss.Addresses {
				addr := &ss.Addresses[i]
				if isValidEndpoint(addr.IP, int(port.Port)) {
					nodeName := defaults.EmptyNodeName
					podName := defaults.EmptyPodName
					if addr.NodeName != nil {
						nodeName = *addr.NodeName
					}
					if addr.TargetRef != nil {
						podName = addr.TargetRef.Name
					}
					endpoint := fmt.Sprintf("%s:%s:%s", nodeName, podName, net.JoinHostPort(addr.IP, strconv.Itoa(int(port.Port))))
					portsToEndpoints[port.Name] = append(portsToEndpoints[port.Name], endpoint)
				}
			}
		}
	}
	return portsToEndpoints
}

// isSessionAffinity return true if this service is using some form of session affinity.
func isSessionAffinity(affinity *affinityPolicy) bool {
	// Should never be empty string, but checking for it to be safe.
	if affinity.affinityType == "" || affinity.affinityType == v1.ServiceAffinityNone {
		return false
	}
	return true
}

// slicesEquiv tests whether two slices are equivalent.  This sorts both slices in-place.
func slicesEquiv(lhs, rhs []string) bool {
	if len(lhs) != len(rhs) {
		return false
	}
	sort.Strings(lhs)
	sort.Strings(rhs)
	return stringslices.Equal(lhs, rhs)
}

// removeSessionAffinityByEndpoint Remove any session affinity records associated to a particular endpoint (for example when a pod goes down).
func removeSessionAffinityByEndpoint(state *balancerState, svcPort proxy.ServicePortName, endpoint string) {
	for _, affinity := range state.affinity.affinityMap {
		if affinity.endpoint == endpoint {
			klog.V(4).InfoS("Removing client from affinityMap for service", "endpoint", affinity.endpoint, "servicePortName", svcPort)
			delete(state.affinity.affinityMap, affinity.clientIP)
		}
	}
}

func isNamespacedNameEqual(dr *istioapi.DestinationRule, namespacedName *types.NamespacedName) bool {
	return dr.Namespace == namespacedName.Namespace && dr.Name == namespacedName.Name
}

func getNamespaceName(dr *istioapi.DestinationRule) string {
	return fmt.Sprintf("%s/%s", dr.Namespace, dr.Name)
}

// getPolicyName gets policy name from a DestinationRule object.
func getPolicyName(dr *istioapi.DestinationRule) string {
	if dr.Spec.TrafficPolicy == nil {
		klog.Errorf("destination rule object %s .Spec.TrafficPolicy is nil", getNamespaceName(dr))
		return ""
	}
	if dr.Spec.TrafficPolicy.LoadBalancer == nil {
		klog.Errorf("destination rule object %s .Spec.TrafficPolicy.LoadBalancer is nil", getNamespaceName(dr))
		return ""
	}
	if dr.Spec.TrafficPolicy.LoadBalancer.LbPolicy == nil {
		klog.Errorf("destination rule object %s .Spec.TrafficPolicy.LoadBalancer.LbPolicy is nil", getNamespaceName(dr))
		return ""
	}
	switch lbPolicy := dr.Spec.TrafficPolicy.LoadBalancer.LbPolicy.(type) {
	case *istiov1alpha3.LoadBalancerSettings_Simple:
		return lbPolicy.Simple.String()
	case *istiov1alpha3.LoadBalancerSettings_ConsistentHash:
		return ConsistentHash
	default:
		klog.Errorf("unsupported load balancer policy %v", lbPolicy)
		return ""
	}
}

// sliceCompare finds the difference between two string slice.
func sliceCompare(src []string, dest []string) ([]string, []string) {
	msrc := make(map[string]byte) // source array set
	mall := make(map[string]byte) // union set
	var set []string              // intersection set

	// 1.Create a map for the source array.
	for _, v := range src {
		msrc[v] = 0
		mall[v] = 0
	}
	// 2.Elements that cannot be stored in the destination array are duplicate elements.
	for _, v := range dest {
		l := len(mall)
		mall[v] = 1
		if l != len(mall) {
			l = len(mall)
		} else {
			set = append(set, v)
		}
	}
	// 3.union - intersection = all variable elements
	for _, v := range set {
		delete(mall, v)
	}
	// 4.Now, mall is a complement set, then we use mall to traverse the source array.
	// The element that can be found is the deleted element, and the element that cannot be found is the added element.
	var added, deleted []string
	for v := range mall {
		_, exist := msrc[v]
		if exist {
			deleted = append(deleted, v)
		} else {
			added = append(added, v)
		}
	}
	return added, deleted
}
