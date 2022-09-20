package util

import (
	"fmt"
	"net"

	"github.com/vishvananda/netlink"
	"k8s.io/klog/v2"
)

// NetifManager copy and update from https://github.com/kubernetes/dns/blob/1.21.0/pkg/netif/netif.go
type NetifManager struct {
	netlink.Handle
	Addrs []*netlink.Addr
}

// NewNetifManager returns a new instance of NetifManager with the ip address set to the provided values
// These ip addresses will be bound to any devices created by this instance.
func NewNetifManager(ips []net.IP) *NetifManager {
	nm := &NetifManager{netlink.Handle{}, nil}
	for _, ip := range ips {
		nm.Addrs = append(nm.Addrs, &netlink.Addr{IPNet: netlink.NewIPNet(ip)})
	}
	return nm
}

// EnsureBridgeDevice checks for the presence of the given bridge device and creates one if it does not exist.
// Returns a boolean to indicate if this device was found and error if any.
func (m *NetifManager) EnsureBridgeDevice(name string) (bool, error) {
	l, err := m.LinkByName(name)
	if err == nil {
		// found bridge device, make sure ip matches. AddrAdd will return error if address exists, will add it otherwise
		for _, addr := range m.Addrs {
			err := m.AddrAdd(l, addr)
			if err != nil {
				klog.V(4).ErrorS(err, "addr %s add error", addr.String())
			}
		}
		return true, nil
	}
	return false, m.AddBridgeDevice(name)
}

// AddBridgeDevice creates a bridge device with the given name. It also binds the ip address of the NetifManager instance
// to this device. This function returns an error if the device exists or if address binding fails.
func (m *NetifManager) AddBridgeDevice(name string) error {
	_, err := m.LinkByName(name)
	if err == nil {
		return fmt.Errorf("link %s exists", name)
	}
	bridge := &netlink.Bridge{
		LinkAttrs: netlink.LinkAttrs{Name: name},
	}
	err = m.LinkAdd(bridge)
	if err != nil {
		return err
	}
	l, _ := m.LinkByName(name)
	for _, addr := range m.Addrs {
		err = m.AddrAdd(l, addr)
		if err != nil {
			return err
		}
	}
	return err
}

// RemoveBridgeDevice deletes the bridge device with the given name.
func (m *NetifManager) RemoveBridgeDevice(name string) error {
	link, err := m.LinkByName(name)
	if err != nil {
		return err
	}
	return m.LinkDel(link)
}

// SetupBridgeDevice setup the bridge device with the given name.
func (m *NetifManager) SetupBridgeDevice(name string) error {
	link, err := m.LinkByName(name)
	if err != nil {
		return err
	}

	return m.LinkSetUp(link)
}

func CreateEdgeMeshDevice(deviceName, deviceIP string) error {
	devIP := net.ParseIP(deviceIP)
	if devIP == nil {
		return fmt.Errorf("failed to parse bridge device IP %s", deviceIP)
	}

	mgr := NewNetifManager([]net.IP{devIP})
	exist, err := mgr.EnsureBridgeDevice(deviceName)
	if exist {
		klog.Infof("bridge device %s already exists", deviceName)
	}
	if err != nil {
		return err
	}

	return mgr.SetupBridgeDevice(deviceName)
}
