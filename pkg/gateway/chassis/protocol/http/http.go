/*
Copyright 2021 The KubeEdge Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package http

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"regexp"
	"strings"

	"github.com/go-chassis/go-chassis/core/invocation"
	apiv1alpha3 "istio.io/api/networking/v1alpha3"
	istioapi "istio.io/client-go/pkg/apis/networking/v1alpha3"
	"k8s.io/klog/v2"

	"github.com/kubeedge/edgemesh/pkg/gateway/chassis/controller"
	"github.com/kubeedge/edgemesh/pkg/gateway/chassis/protocol"
	"github.com/kubeedge/edgemesh/pkg/gateway/chassis/protocol/tcp"
)

func init() {
	http := &HTTP{}
	http.Register()
}

// HTTP http
type HTTP struct {
	Conn           net.Conn
	VirtualService *istioapi.VirtualService
	SvcName        string
	SvcNamespace   string
	Port           int
	Req            *http.Request
	Resp           *http.Response
}

// register protocol name
func (p *HTTP) Register() {
	protocol.RegisterProtocols = append(protocol.RegisterProtocols, "http")
}

// Process process
func (p *HTTP) Process() {
	for {
		// parse http request
		req, err := http.ReadRequest(bufio.NewReader(p.Conn))
		if err != nil || err == io.EOF {
			if err = p.Conn.Close(); err != nil {
				klog.Errorf("close conn err: %v", err)
			}
			return
		}

		// route
		if p.VirtualService != nil {
			err = p.route(req.RequestURI)
			if err != nil {
				klog.Errorf("route by http request uri err: %v", err)
				if err = p.Conn.Close(); err != nil {
					klog.Errorf("close conn err: %v", err)
				}
				return
			}
		}

		// http fallback to tcp
		reqBytes, err := httpRequestToBytes(req)
		if err != nil {
			klog.Errorf("transforms http request to bytes err: %v", err)
			if err = p.Conn.Close(); err != nil {
				klog.Errorf("close conn err: %v", err)
			}
			return
		}

		httpToTcp := &tcp.TCP{
			Conn:         p.Conn,
			SvcNamespace: p.SvcNamespace,
			SvcName:      p.SvcName,
			Port:         p.Port,
			UpgradeReq:   reqBytes,
		}
		httpToTcp.Process()
	}
}

func uriMatch(sm *apiv1alpha3.StringMatch, reqURI string) bool {
	if (sm.GetExact() != "" && sm.GetExact() == reqURI) || (sm.GetPrefix() != "" && strings.HasPrefix(reqURI, sm.GetPrefix())) {
		return true
	}
	if rg := sm.GetRegex(); rg != "" {
		uriPattern, err := regexp.Compile(rg)
		if err != nil {
			klog.Errorf("string match regex compile err: %v", err)
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
				svcName := httpRoute.Route[0].Destination.Host
				svcNamespace := p.VirtualService.Namespace
				// find a service
				key := fmt.Sprintf("%s.%s", svcNamespace, svcName)
				if _, err := controller.APIConn.GetSvcLister().Services(svcNamespace).Get(svcName); err != nil {
					return fmt.Errorf("service bound to the destination %s does not exist, reason: %v", key, err)
				}
				klog.Infof("destination svc is %s", key)
				p.SvcName = svcName
				p.SvcNamespace = svcNamespace
				p.Port = int(httpRoute.Route[0].Destination.Port.Number)
				return nil
			}
		}
	}
	return fmt.Errorf("no match svc found for uri %s", requestURI)
}

// responseCallback process invocation response
func (p *HTTP) responseCallback(data *invocation.Response) error {
	var errMsg string
	if data.Err != nil {
		if err := p.responseUnavailable(data.Err.Error()); err != nil {
			return err
		}
		return data.Err
	}
	if data.Result == nil {
		errMsg = "httpserver response nil"
		if err := p.responseUnavailable(errMsg); err != nil {
			return err
		}
		return fmt.Errorf(errMsg)
	}
	resp, ok := data.Result.(*http.Response)
	if !ok {
		errMsg = "invalid http response"
		if err := p.responseUnavailable(errMsg); err != nil {
			return err
		}
		return fmt.Errorf(errMsg)
	}
	respBytes, err := httpResponseToBytes(resp)
	if err != nil {
		errMsg = "http response to bytes failed"
		if err := p.responseUnavailable(errMsg); err != nil {
			return err
		}
		return fmt.Errorf(errMsg)
	}
	// write data to http conn
	_, err = p.Conn.Write(respBytes)
	if err != nil {
		return err
	}
	return nil
}

// responseUnavailable return 503 to http conn
func (p *HTTP) responseUnavailable(errMsg string) error {
	resp := &http.Response{
		Status:     fmt.Sprintf("%d %s", http.StatusServiceUnavailable, errMsg),
		StatusCode: http.StatusServiceUnavailable,
		Proto:      p.Req.Proto,
		Request:    p.Req,
		Header:     make(http.Header),
	}
	respBytes, err := httpResponseToBytes(resp)
	if err != nil {
		return err
	}
	_, err = p.Conn.Write(respBytes)
	if err != nil {
		return err
	}
	return nil
}

// httpResponseToBytes transforms http.Response to bytes
func httpResponseToBytes(resp *http.Response) ([]byte, error) {
	buf := new(bytes.Buffer)
	if resp == nil {
		return nil, fmt.Errorf("http response nil")
	}
	err := resp.Write(buf)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// httpRequestToBytes transforms http.Request to bytes
func httpRequestToBytes(req *http.Request) ([]byte, error) {
	buf := new(bytes.Buffer)
	if req == nil {
		return nil, fmt.Errorf("http request nil")
	}
	err := req.Write(buf)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
