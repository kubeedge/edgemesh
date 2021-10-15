package util

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/klog/v2"
)

const (
	timeout = 5 * time.Second
	retry   = 3
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

func FetchPublicIP() string {
	Client := http.Client{
		Timeout: timeout,
	}
	var resp *http.Response
	var err error
	for i := 0; i < retry; i++ {
		resp, err = Client.Get("https://ifconfig.me")
		if err == nil {
			break
		}
	}
	if err != nil {
		klog.Errorf("fetch public ip failed, %v", err)
		return ""
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		klog.Errorf("fetch public ip failed, %v", err)
		return ""
	}
	return string(body)
}
