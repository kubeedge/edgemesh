package util

import (
	"fmt"
	"net"

	"k8s.io/dns/pkg/netif"
)

func CreateDummyDevice(deviceName, deviceIP string) error {
	devIP := net.ParseIP(deviceIP)
	if devIP == nil {
		return fmt.Errorf("failed to parse dummy device IP %s", deviceIP)
	}

	mgr := netif.NewNetifManager([]net.IP{devIP})
	exist, err := mgr.EnsureDummyDevice(deviceName)
	if err != nil {
		return err
	}

	if exist {
		if err = mgr.RemoveDummyDevice(deviceName); err != nil {
			return err
		}
	}

	if err = mgr.AddDummyDevice(deviceName); err != nil {
		return err
	}

	// dummy device up
	link, err := mgr.LinkByName(deviceName)
	if err != nil {
		return err
	}

	return mgr.LinkSetUp(link)
}
