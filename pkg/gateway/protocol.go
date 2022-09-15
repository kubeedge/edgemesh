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

	apiv1alpha3 "istio.io/api/networking/v1alpha3"
	istioapi "istio.io/client-go/pkg/apis/networking/v1alpha3"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"

	netutil "github.com/kubeedge/edgemesh/pkg/util/net"
)

// Protocol protocol
type Protocol interface {
	Process()
}

// HTTP http
type HTTP struct {
	Conn           net.Conn
	VirtualService *istioapi.VirtualService
	SvcName        string
	SvcNamespace   string
	SvcPort        int
}

func (p *HTTP) Process() {
	for {
		// parse http request
		req, err := http.ReadRequest(bufio.NewReader(p.Conn))
		if err != nil {
			if errors.Is(err, io.EOF) {
				klog.Infof("read http request EOF")
				err = p.Conn.Close()
				if err != nil {
					klog.Errorf("close conn err: %v", err)
				}
				return
			}
			klog.Errorf("read http request err: %v", err)
			err = p.Conn.Close()
			if err != nil {
				klog.Errorf("close conn err: %v", err)
			}
			return
		}

		// route
		if p.VirtualService != nil {
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

		reqBytes, err := netutil.HttpRequestToBytes(req)
		if err != nil {
			klog.Errorf("request convert to bytes error: %v", err)
			err = p.Conn.Close()
			if err != nil {
				klog.Errorf("close conn err: %v", err)
			}
			return
		}

		httpToTcp := &TCP{
			Conn:         p.Conn,
			SvcNamespace: p.SvcNamespace,
			SvcName:      p.SvcName,
			SvcPort:      p.SvcPort,
			UpgradeReq:   reqBytes,
		}
		httpToTcp.Process()
	}
}

func uriMatch(sm *apiv1alpha3.StringMatch, reqURI string) bool {
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

type TCP struct {
	Conn         net.Conn
	SvcName      string
	SvcNamespace string
	SvcPort      int
	UpgradeReq   []byte
}

func (p *TCP) Process() {
	// find a service
	svcName := types.NamespacedName{Namespace: p.SvcNamespace, Name: p.SvcName}
	svcPort, ok := internalLoadBalancer.GetServicePortName(svcName, p.SvcPort)
	if !ok {
		klog.Errorf("destination service %s not found in cache", svcName)
		return
	}
	klog.Infof("destination service is %s", svcPort)

	klog.V(3).InfoS("Accepted TCP connection from remote", "remoteAddress", p.Conn.RemoteAddr(), "localAddress", p.Conn.LocalAddr())
	outConn, err := internalLoadBalancer.TryConnectEndpoints(svcPort, p.Conn.RemoteAddr(), "tcp", p.Conn)
	if err != nil {
		klog.ErrorS(err, "Failed to connect to balancer")
		p.Conn.Close()
		return
	}

	if len(p.UpgradeReq) > 0 {
		_, err = outConn.Write(p.UpgradeReq)
		if err != nil {
			klog.ErrorS(err, "Failed to write UpgradeReq")
			p.Conn.Close()
			return
		}
	}

	go netutil.ProxyConn(p.Conn, outConn)
}
