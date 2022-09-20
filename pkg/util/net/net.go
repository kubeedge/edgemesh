package util

import (
	"fmt"
	"net"
)

const IPv4Mask = 4

// GetInterfaceIP get net interface ipv4 address
func GetInterfaceIP(name string) (net.IP, error) {
	ifi, err := net.InterfaceByName(name)
	if err != nil {
		return nil, err
	}
	addrs, err := ifi.Addrs()
	if err != nil {
		return nil, err
	}
	for _, addr := range addrs {
		ip, ipn, err := net.ParseCIDR(addr.String())
		if err != nil {
			return nil, err
		}

		if len(ipn.Mask) == IPv4Mask {
			return ip, nil
		}
	}
	return nil, fmt.Errorf("no ip of version 4 found for interface %s", name)
}
