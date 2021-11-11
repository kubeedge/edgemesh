package dns

import (
	"io/ioutil"
	"net"
	"strings"
	"time"

	mdns "github.com/miekg/dns"
	"k8s.io/klog/v2"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/edgemesh/agent/pkg/dns/controller"
	"github.com/kubeedge/edgemesh/common/util"
)

const hostResolv = "/etc/resolv.conf"

type handler struct{}

func (h *handler) ServeDNS(w mdns.ResponseWriter, r *mdns.Msg) {
	msg := mdns.Msg{}
	msg.SetReply(r)
	switch r.Question[0].Qtype {
	case mdns.TypeA:
		msg.Authoritative = true
		domain := msg.Question[0].Name
		address, ok := lookup(domain)
		if ok {
			msg.Answer = append(msg.Answer, &mdns.A{
				Hdr: mdns.RR_Header{Name: domain, Rrtype: mdns.TypeA, Class: mdns.ClassINET, Ttl: 60},
				A:   net.ParseIP(address),
			})
		} else {
			return
		}
	}
	if err := w.WriteMsg(&msg); err != nil {
		klog.Errorf("dns response send error: %v", err)
	}
}

func (dns *EdgeDNS) Run() {
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

	dns.Server.Handler = &handler{}
	if err := dns.Server.ListenAndServe(); err != nil {
		klog.Errorf("dns server serve error: %v", err)
	}
}

// lookup confirms if the service exists
func lookup(serviceURL string) (ip string, exist bool) {
	// Here serviceURL is a domain name which has at least a "." suffix. So here we need trim it.
	if strings.HasSuffix(serviceURL, ".") {
		serviceURL = strings.TrimSuffix(serviceURL, ".")
	}
	name, namespace := util.SplitServiceKey(serviceURL)
	ip, err := controller.APIConn.GetSvcIP(namespace, name)
	if err != nil {
		klog.Errorf("service reverse clusterIP error: %v", err)
		return "", false
	}
	klog.Infof("dns server parse %s ip %s", serviceURL, ip)
	return ip, true
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
