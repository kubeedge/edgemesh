package util

import (
	"fmt"
	"net"
	"os"
	"strings"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
)

// SplitServiceKey splits service name
func SplitServiceKey(key string) (name, namespace string) {
	sets := strings.Split(key, ".")
	if len(sets) >= 2 {
		return sets[0], sets[1]
	}
	ns := os.Getenv("POD_NAMESPACE")
	if ns == "" {
		ns = "default"
	}
	if len(sets) == 1 {
		return sets[0], ns
	}
	return key, ns
}

// GetInterfaceIP get net interface ipv4 address
func GetInterfaceIP(name string) (net.IP, error) {
	ifi, err := net.InterfaceByName(name)
	if err != nil {
		return nil, err
	}
	addrs, _ := ifi.Addrs()
	for _, addr := range addrs {
		if ip, ipn, _ := net.ParseCIDR(addr.String()); len(ipn.Mask) == 4 {
			return ip, nil
		}
	}
	return nil, fmt.Errorf("no ip of version 4 found for interface %s", name)
}

// GetPodsSelector use the selector to obtain the backend pods bound to the service
func GetPodsSelector(svc *v1.Service) labels.Selector {
	selector := labels.NewSelector()
	for k, v := range svc.Spec.Selector {
		r, _ := labels.NewRequirement(k, selection.Equals, []string{v})
		selector = selector.Add(*r)
	}
	return selector
}
