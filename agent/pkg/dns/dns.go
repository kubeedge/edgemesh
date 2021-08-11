package dns

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strings"
	"time"

	"k8s.io/klog/v2"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/edgemesh/agent/pkg/dns/controller"
	"github.com/kubeedge/edgemesh/common/util"
)

const hostResolv = "/etc/resolv.conf"

func (dns *EdgeDNS) Run() {
	defer dns.DNSConn.Close()

	// ensure /etc/resolv.conf have dns nameserver
	go func() {
		dns.ensureResolvForHost()
		ticker := time.NewTicker(time.Minute)
		for {
			select {
			case <-ticker.C:
				dns.ensureResolvForHost()
			case <-beehiveContext.Done():
				dns.cleanResolvForHost()
				return
			}
		}
	}()

	// start dns server
	for {
		req := make([]byte, bufSize)
		n, from, err := dns.DNSConn.ReadFromUDP(req)
		if err != nil || n <= 0 {
			klog.Errorf("dns server read from udp error: %v", err)
			continue
		}

		que, err := parseDNSQuery(req[:n])
		if err != nil {
			continue
		}

		que.from = from

		rsp, err := dns.recordHandle(que, req[:n])
		if err != nil {
			klog.Warningf("resolve dns: %v", err)
			continue
		}
		if _, err := dns.DNSConn.WriteTo(rsp, from); err != nil {
			klog.Warningf("failed to write: %v", err)
		}
	}
}

// recordHandle returns the answer for the dns question
func (dns *EdgeDNS) recordHandle(que *dnsQuestion, req []byte) (rsp []byte, err error) {
	var exist bool
	var ip string
	// qType should be 1 for ipv4
	if que.name != nil && que.qType == aRecord {
		domainName := string(que.name)
		exist, ip = dns.lookup(domainName)
	}

	if !exist || que.event == eventUpstream {
		// if this listener doesn't belongs to this cluster
		go dns.getFromRealDNS(req, que.from)
		return rsp, fmt.Errorf("get from real dns")
	}

	address := net.ParseIP(ip).To4()
	if address == nil {
		que.event = eventNxDomain
	}
	// gen
	pre := modifyRspPrefix(que)
	rsp = append(rsp, pre...)
	if que.event != eventNothing {
		return rsp, nil
	}
	// create a deceptive resp, if no error
	dnsAns := &dnsAnswer{
		name:    que.name,
		qType:   que.qType,
		qClass:  que.qClass,
		ttl:     ttl,
		dataLen: uint16(len(address)),
		addr:    address,
	}
	ans := dnsAns.getAnswer()
	rsp = append(rsp, ans...)

	return rsp, nil
}

// lookup confirms if the service exists
func (dns *EdgeDNS) lookup(serviceURL string) (exist bool, ip string) {
	name, namespace := util.SplitServiceKey(serviceURL)
	ip, err := controller.APIConn.GetSvcIP(namespace, name)
	if err != nil {
		klog.Errorf("service reverse clusterIP error: %v", err)
		return false, ""
	}
	klog.Infof("dns server parse %s ip %s", serviceURL, ip)
	return true, ip
}

// getFromRealDNS returns a dns response from real dns servers
func (dns *EdgeDNS) getFromRealDNS(req []byte, from *net.UDPAddr) {
	rsp := make([]byte, 0)
	ips, err := dns.parseNameServer()
	if err != nil {
		klog.Errorf("parse nameserver err: %v", err)
		return
	}

	laddr := &net.UDPAddr{
		IP:   net.IPv4zero,
		Port: 0,
	}

	// get from real dns servers
	for _, ip := range ips {
		raddr := &net.UDPAddr{
			IP:   ip,
			Port: 53,
		}
		conn, err := net.DialUDP("udp", laddr, raddr)
		if err != nil {
			continue
		}
		defer conn.Close()
		_, err = conn.Write(req)
		if err != nil {
			continue
		}
		if err = conn.SetReadDeadline(time.Now().Add(time.Minute)); err != nil {
			continue
		}
		var n int
		buf := make([]byte, bufSize)
		n, err = conn.Read(buf)
		if err != nil {
			continue
		}

		if n > 0 {
			rsp = append(rsp, buf[:n]...)
			if _, err = dns.DNSConn.WriteToUDP(rsp, from); err != nil {
				klog.Errorf("failed to wirte to udp, err: %v", err)
				continue
			}
			break
		}
	}
}

// parseNameServer gets all real nameservers from the resolv.conf
func (dns *EdgeDNS) parseNameServer() ([]net.IP, error) {
	file, err := os.Open("/etc/resolv.conf")
	if err != nil {
		return nil, fmt.Errorf("error opening /etc/resolv.conf: %v", err)
	}
	defer file.Close()

	scan := bufio.NewScanner(file)
	scan.Split(bufio.ScanLines)

	ip := make([]net.IP, 0)

	for scan.Scan() {
		serverString := scan.Text()
		if strings.Contains(serverString, "nameserver") {
			tmpString := strings.Replace(serverString, "nameserver", "", 1)
			nameserver := strings.TrimSpace(tmpString)
			sip := net.ParseIP(nameserver)
			if sip != nil && !sip.Equal(dns.ListenIP) {
				ip = append(ip, sip)
			}
		}
	}
	if len(ip) == 0 {
		return nil, fmt.Errorf("there is no nameserver in /etc/resolv.conf")
	}
	return ip, nil
}

// ensureResolvForHost adds edgemesh dns server to the head of /etc/resolv.conf
func (dns *EdgeDNS) ensureResolvForHost() {
	bs, err := ioutil.ReadFile(hostResolv)
	if err != nil {
		klog.Errorf("read file %s err: %v", hostResolv, err)
		return
	}

	resolv := strings.Split(string(bs), "\n")
	if resolv == nil {
		nameserver := "nameserver " + dns.ListenIP.String()
		if err := ioutil.WriteFile(hostResolv, []byte(nameserver), 0600); err != nil {
			klog.Errorf("write file %s err: %v", hostResolv, err)
		}
		return
	}

	configured := false
	dnsIdx := 0
	startIdx := 0
	for idx, item := range resolv {
		if strings.Contains(item, dns.ListenIP.String()) {
			configured = true
			dnsIdx = idx
			break
		}
	}
	for idx, item := range resolv {
		if strings.Contains(item, "nameserver") {
			startIdx = idx
			break
		}
	}
	if configured {
		if dnsIdx != startIdx && dnsIdx > startIdx {
			nameserver := sortNameserver(resolv, dnsIdx, startIdx)
			if err := ioutil.WriteFile(hostResolv, []byte(nameserver), 0600); err != nil {
				klog.Errorf("failed to write file %s, err: %v", hostResolv, err)
				return
			}
		}
		return
	}

	nameserver := ""
	for idx := 0; idx < len(resolv); {
		if idx == startIdx {
			startIdx = -1
			nameserver = nameserver + "nameserver " + dns.ListenIP.String() + "\n"
			continue
		}
		nameserver = nameserver + resolv[idx] + "\n"
		idx++
	}

	if err := ioutil.WriteFile(hostResolv, []byte(nameserver), 0600); err != nil {
		klog.Errorf("failed to write file %s, err: %v", hostResolv, err)
		return
	}
}

func sortNameserver(resolv []string, dnsIdx, startIdx int) string {
	nameserver := ""
	idx := 0
	for ; idx < startIdx; idx++ {
		nameserver = nameserver + resolv[idx] + "\n"
	}
	nameserver = nameserver + resolv[dnsIdx] + "\n"

	for idx = startIdx; idx < len(resolv); idx++ {
		if idx == dnsIdx {
			continue
		}
		nameserver = nameserver + resolv[idx] + "\n"
	}

	return nameserver
}

// cleanResolvForHost delete edgemesh dns server from the head of /etc/resolv.conf
func (dns *EdgeDNS) cleanResolvForHost() {
	bs, err := ioutil.ReadFile(hostResolv)
	if err != nil {
		klog.Warningf("read file %s err: %v", hostResolv, err)
	}

	resolv := strings.Split(string(bs), "\n")
	if resolv == nil {
		return
	}
	nameserver := ""
	for _, item := range resolv {
		if strings.Contains(item, dns.ListenIP.String()) || item == "" {
			continue
		}
		nameserver = nameserver + item + "\n"
	}
	if err := ioutil.WriteFile(hostResolv, []byte(nameserver), 0600); err != nil {
		klog.Errorf("failed to write nameserver to file %s, err: %v", hostResolv, err)
	}
}
