package gateway

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"regexp"
	"strings"

	networkingv1alpha3 "istio.io/api/networking/v1alpha3"
	istioapi "istio.io/client-go/pkg/apis/networking/v1alpha3"
	"k8s.io/klog/v2"
)

// HTTP http
type HTTP struct {
	Conn           net.Conn
	VirtualService *istioapi.VirtualService
	SvcName        string
	SvcNamespace   string
	SvcPort        int
}

// Process process
func (p *HTTP) Process() {
	for {
		// parse http request
		req, err := http.ReadRequest(bufio.NewReader(p.Conn))
		if err != nil {
			if !errors.Is(err, io.EOF) {
				klog.Errorf("read http request err: %v", err)
			}
			err = p.Conn.Close()
			if err != nil {
				klog.Errorf("close conn err: %v", err)
			}
			return
		}

		// route
		if p.VirtualService != nil {
			err = p.checkHost(req.Host)
			if err != nil {
				klog.Errorf("check http request host err: %v", err)
				err = p.Conn.Close()
				if err != nil {
					klog.Errorf("close conn err: %v", err)
				}
				return
			}
			err = p.route(req.RequestURI)
			if err != nil {
				klog.Errorf("route by http request uri err: %v", err)
				err = p.Conn.Close()
				if err != nil {
					klog.Errorf("close conn err: %v", err)
				}
				return
			}
		}

		// passthrough
		httpToTcp := &TCP{
			Conn:         p.Conn,
			SvcNamespace: p.SvcNamespace,
			SvcName:      p.SvcName,
			SvcPort:      p.SvcPort,
			UpgradeReq:   req,
		}
		httpToTcp.Process()
	}
}

func uriMatch(sm *networkingv1alpha3.StringMatch, reqURI string) bool {
	isRoot := false
	if reqURI == "/" {
		isRoot = true
	}
	prefix := sm.GetPrefix()
	exact := sm.GetExact()
	regex := sm.GetRegex()
	if !isRoot {
		reqURI = strings.TrimLeft(reqURI, "/")
		prefix = strings.TrimLeft(prefix, "/")
		exact = strings.TrimLeft(exact, "/")
		regex = strings.TrimLeft(regex, "/")
	}

	if (exact != "" && exact == reqURI) || (prefix != "" && strings.HasPrefix(reqURI, prefix)) {
		return true
	}
	if regex != "" {
		uriPattern, err := regexp.Compile(regex)
		if err != nil {
			klog.Errorf("string match regex compile err: %v", err)
			return false
		}
		if uriPattern.Match([]byte(reqURI)) {
			return true
		}
	}
	return false
}

// route updates service meta
func (p *HTTP) route(requestURI string) error {
	if p.VirtualService == nil {
		return errors.New("virtual service nil")
	}
	for _, httpRoute := range p.VirtualService.Spec.Http {
		for _, httpMatchRequest := range httpRoute.Match {
			if ok := uriMatch(httpMatchRequest.Uri, requestURI); ok && len(httpRoute.Route) > 0 {
				p.SvcName = httpRoute.Route[0].Destination.Host
				p.SvcNamespace = p.VirtualService.Namespace
				p.SvcPort = int(httpRoute.Route[0].Destination.Port.Number)
				return nil
			}
		}
	}
	return fmt.Errorf("no match svc found for uri %s", requestURI)
}

func (p *HTTP) checkHost(requestHost string) error {
	if p.VirtualService == nil {
		return errors.New("virtual service nil")
	}

	vsHosts := p.VirtualService.Spec.GetHosts()
	if len(vsHosts) < 1 {
		return errors.New("virtual service no hosts")
	}

	// check if allow all domain
	allowAll := false
	for _, vsHost := range vsHosts {
		if vsHost == "*" {
			allowAll = true
			break
		}
	}

	if allowAll {
		return nil
	}

	// parse reqHost to domain
	hostElem := strings.Split(requestHost, ":")
	if len(hostElem) < 1 {
		return fmt.Errorf("invalid request host %s", requestHost)
	}
	host := hostElem[0]
	if host == "" {
		return fmt.Errorf("invalid request host %s", requestHost)
	}

	// allow all ip-address-host
	hostIP := net.ParseIP(host)
	if hostIP != nil {
		return nil
	}

	// check if host in domain list
	for _, vsHost := range vsHosts {
		if host == vsHost {
			return nil
		}
	}

	return fmt.Errorf("`%s` not in domain list", host)
}
