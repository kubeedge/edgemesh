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

	meshConstants "github.com/kubeedge/edgemesh/common/constants"
)

const (
	timeout     = 5 * time.Second
	retry       = 3
	ifconfigURL = "https://ifconfig.me"
)

// SplitServiceKey splits service name
func SplitServiceKey(key string) (name, namespace string) {
	sets := strings.Split(key, ".")
	if len(sets) >= 2 {
		return sets[0], sets[1]
	}
	return key, "default"
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
		resp, err = Client.Get(ifconfigURL)
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

var nodeName string

func FetchNodeName() string {
	if nodeName != "" {
		return nodeName
	}

	var isExist bool
	nodeName, isExist = os.LookupEnv(meshConstants.MY_NODE_NAME)
	if !isExist {
		klog.Exitf("env %s not exist", meshConstants.MY_NODE_NAME)
	}
	return nodeName
}

func IsNotFoundError(err error) bool {
	return strings.Contains(err.Error(), "not found")
}
