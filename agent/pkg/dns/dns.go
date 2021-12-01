package dns

import (
	"net"
	"strings"

	mdns "github.com/miekg/dns"
	"k8s.io/klog/v2"

	"github.com/kubeedge/edgemesh/agent/pkg/dns/controller"
	"github.com/kubeedge/edgemesh/common/util"
)

func (dns *EdgeDNS) Run() {
	dns.Server.Handler = &handler{}
	if err := dns.Server.ListenAndServe(); err != nil {
		klog.Errorf("dns server serve error: %v", err)
	}
}

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

// lookup confirms if the service exists
func lookup(serviceURL string) (ip string, exist bool) {
	// Here serviceURL is a domain name which has at least a "." suffix. So here we need trim it.
	serviceURL = strings.TrimSuffix(serviceURL, ".")
	name, namespace := util.SplitServiceKey(serviceURL)
	ip, err := controller.APIConn.GetSvcIP(namespace, name)
	if err != nil {
		klog.Errorf("service `%s.%s` reverse clusterIP error: %v", name, namespace, err)
		return "", false
	}
	klog.Infof("dns server parse %s ip %s", serviceURL, ip)
	return ip, true
}
