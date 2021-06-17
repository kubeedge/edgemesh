package util

import (
	"net"
	"strings"

	"k8s.io/klog/v2"

	"github.com/kubeedge/edgemesh/pkg/networking/edgegateway/config"
)

// GetIPsNeedListen get Ips need listen
func GetIPsNeedListen() ([]net.IP, error) {
	ips := make([]net.IP, 0)
	// step 1 :network cards info by env "NIC"
	netCards := config.Config.NIC
	// case 1: env "NIC" not set or is "*"
	if netCards == "" || netCards == "*" {
		klog.Warningf("NIC is empty or *, listen all netcards.")
		var err error
		ips, err = GetAllIPs()
		if err != nil {
			klog.Errorf("GetAllIPs failed, err: %v.", err)
			return nil, err
		}
		klog.Infof("ips is %+v.", ips)
	} else {
		// case 2: env "NIC" is set. get all network cards, for example "lo,eth0"
		netList := strings.Split(netCards, ",")
		for _, v := range netList {
			ipsByName, err := GetIPsByName(v)
			if err != nil {
				klog.Errorf("GetIPsByName failed, err: %v.", err)
				continue
			}
			ips = append(ips, ipsByName...)
		}
		klog.Infof("ips is %+v.", ips)
	}

	// step 2 :get ips include. by env "INCLUDE_IP"
	includeIPs := config.Config.IncludeIP
	if includeIPs == "" || includeIPs == "*" {
		klog.Infof("INCLUDE_IP val is empty or *, include all ips.")
	} else {
		includeIPList := strings.Split(includeIPs, ",")
		incIPList, incIPSegmentList := getIPAndSegment(includeIPList)
		ips = includeIPsFromIPs(ips, incIPList, incIPSegmentList)
	}

	// step 3: get ips exclude. by env "EXCLUDE_IP"
	excludeIPs := config.Config.ExcludeIP
	// case 1: env "EXCLUDE_IP" is empty, not exclude any ip.
	if excludeIPs == "" {
		klog.Infof("EXCLUDE_IP val is empty, not exclude ip.")
		return ips, nil
	}
	// case 2, env EXCLUDE_IP is set.
	excludeIPList := strings.Split(excludeIPs, ",")
	ipList, ipSegmentList := getIPAndSegment(excludeIPList)
	excludeIPres := excludeIPsFromIPs(ips, ipList)
	res := excludeIPSegmentsFromIPs(excludeIPres, ipSegmentList)
	return res, nil
}

// GetAllIPs get all IPs
func GetAllIPs() ([]net.IP, error) {
	ips := make([]net.IP, 0)

	interfaces, err := net.Interfaces()
	if err != nil {
		klog.Errorf("get Interfaces failed, err: %v", err)
		return nil, err
	}

	for _, i := range interfaces {
		res, err := GetIPsByName(i.Name)
		if err != nil {
			klog.Errorf("GetIPsByName failed, err: %v", err)
			return nil, err
		}
		ips = append(ips, res...)
	}
	return ips, nil
}

// GetIPsByName get IPs by name
func GetIPsByName(name string) ([]net.IP, error) {
	ips := make([]net.IP, 0)
	interfaceInfo, err := net.InterfaceByName(name)
	if err != nil {
		klog.Errorf("get InterfaceByName failed, err: %v", err)
		return nil, err
	}
	addresses, err := interfaceInfo.Addrs()
	if err != nil {
		klog.Errorf("get addrs failed, err: %v", err)
		return nil, err
	}
	for _, v := range addresses {
		if ip, ipnet, err := net.ParseCIDR(v.String()); err == nil && len(ipnet.Mask) == 4 {
			ips = append(ips, ip)
		}
	}

	return ips, nil
}

// get ipList and ipSegmentList.
func getIPAndSegment(IPList []string) ([]string, []string) {
	klog.Infof("start getIPAndSegment, IPList is %+v.", IPList)
	ipList := make([]string, 0)
	ipSegmentList := make([]string, 0)
	for _, val := range IPList {
		if strings.Contains(val, "/") {
			ipSegmentList = append(ipSegmentList, val)
			continue
		}
		ipList = append(ipList, val)
	}
	klog.Infof("after getIPAndSegment, ipList is %+v. ipSegmentList is %v.", ipList, ipSegmentList)
	return ipList, ipSegmentList
}

func excludeIPsFromIPs(allIPList []net.IP, excludeIPList []string) []net.IP {
	klog.Infof("start excludeIPsFromIPs, allIPList is %+v.", allIPList)
	for i := 0; i < len(allIPList); i++ {
		for _, ex := range excludeIPList {
			klog.Infof("allIPList[i] : %s, excludeIP : %s.", allIPList[i].String(), ex)
			if allIPList[i].String() == ex {
				klog.Infof("allIPList[i] and excludeIP equal.")
				allIPList = append(allIPList[:i], allIPList[i+1:]...)
				i--
				break
			}
		}
	}
	klog.Infof("after excludeIPsFromIPs, allIPList is %+v.", allIPList)
	return allIPList
}

func excludeIPSegmentsFromIPs(allIPList []net.IP, excludeIPSegmentList []string) []net.IP {
	klog.Infof("start excludeIPSegmentsFromIPs, allIPList is %+v.", allIPList)
	for i := 0; i < len(allIPList); i++ {
		for _, ex := range excludeIPSegmentList {
			_, ipnet, err := net.ParseCIDR(ex)
			if err != nil {
				klog.Errorf("ParseCIDR %s failed, err: %v", ex, err)
				continue
			}
			if ipnet.Contains(allIPList[i]) {
				allIPList = append(allIPList[:i], allIPList[i+1:]...)
				i--
				break
			}
		}
	}
	klog.Infof("after excludeIPSegmentsFromIPs, allIPList is %+v.", allIPList)
	return allIPList
}

func includeIPsFromIPs(allIPList []net.IP, includeIPList []string, includeIPSegmentList []string) []net.IP {
	klog.Infof("start includeIPsFromIPs, allIPList is %+v.", allIPList)
	res := make([]net.IP, 0)
	for i := 0; i < len(allIPList); i++ {
		match := false
		for _, val := range includeIPList {
			if allIPList[i].String() == val {
				res = append(res, allIPList[i])
				match = true
				break
			}
		}
		// not add the same ip again.
		if match {
			continue
		}

		for _, val := range includeIPSegmentList {
			_, ipnet, err := net.ParseCIDR(val)
			if err != nil {
				klog.Errorf("ParseCIDR %s failed, err: %v", val, err)
				continue
			}
			if ipnet.Contains(allIPList[i]) {
				res = append(res, allIPList[i])
				break
			}
		}
	}
	klog.Infof("after includeIPsFromIPs, res is %+v.", res)
	return res
}
