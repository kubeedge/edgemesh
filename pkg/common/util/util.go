package util

import (
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"k8s.io/klog/v2"
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

// TODO remove
var nodeName string

func FetchNodeName() string {
	if nodeName != "" {
		return nodeName
	}

	var isExist bool
	nodeName, isExist = os.LookupEnv("NODE_NAME")
	if !isExist {
		klog.Exitf("env %s not exist", "NODE_NAME")
	}
	return nodeName
}
