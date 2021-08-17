package dns

import (
	"net"

	"k8s.io/klog/v2"

	"github.com/kubeedge/edgemesh/agent/pkg/dns/controller"
	"github.com/kubeedge/edgemesh/common/util"

	mdns "github.com/miekg/dns"
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
			msg.Answer = append(msg.Answer, &dns.A{
				Hdr: dns.RR_Header{Name: domain, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 60},
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
	dns.Server.Handler = &handler{}
	if err := dns.Server.ListenAndServe(); err != nil {
		klog.Errorf("dns server serve error: %v", err)
	}
}

// lookup confirms if the service exists
func lookup(serviceURL string) (ip string, exist bool) {
	name, namespace := util.SplitServiceKey(serviceURL)
	ip, err := controller.APIConn.GetSvcIP(namespace, name)
	if err != nil {
		klog.Errorf("service reverse clusterIP error: %v", err)
		return "", false
	}
	klog.Infof("dns server parse %s ip %s", serviceURL, ip)
	return ip, true
}
