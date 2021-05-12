package server

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	istioapi "istio.io/client-go/pkg/apis/networking/v1alpha3"
	"net"
	"reflect"
	"sync"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/klog/v2"

	"github.com/kubeedge/edgemesh/pkg/controller"
	"github.com/kubeedge/edgemesh/pkg/networking/trafficplugin/protocol"
	"github.com/kubeedge/edgemesh/pkg/networking/trafficplugin/protocol/http"
	"github.com/kubeedge/edgemesh/pkg/networking/trafficplugin/protocol/tcp"
)

// Server is gateway server
type Server struct {
	listener *net.TCPListener
	stop     chan interface{}
	wg       sync.WaitGroup
	options  *Options
}

// Options options
type Options struct {
	// ingress
	Exposed        bool
	GwName         string
	Namespace      string
	Hosts          []string
	Protocol       string
	CredentialName string
	MinVersion     uint16
	MaxVersion     uint16
	CipherSuites   []uint16
}

// NewServer new server
func NewServer(ip net.IP, port int, opts *Options) (*Server, error) {
	laddr := &net.TCPAddr{
		IP:   ip,
		Port: port,
	}
	ln, err := net.ListenTCP("tcp", laddr)
	if err != nil {
		return nil, err
	}
	s := &Server{
		listener: ln,
		stop:     make(chan interface{}),
		options:  opts,
	}
	s.wg.Add(1)
	go s.serve()
	return s, nil
}

func (srv *Server) serve() {
	defer srv.wg.Done()

	for {
		conn, err := srv.listener.Accept()
		if err != nil {
			select {
			case _, isClosed := <-srv.stop:
				if !isClosed {
					klog.Errorf("server stop to serve")
				}
				return
			default:
				klog.Warningf("get conn error: %v", err)
			}
		} else {
			// tls
			if srv.options.CredentialName != "" {
				klog.Infof("tls required")
				key := fmt.Sprintf("%s.%s", srv.options.Namespace, srv.options.CredentialName)
				s, err := controller.GetSecretLister().Secrets(srv.options.Namespace).Get(srv.options.CredentialName)
				if err != nil {
					klog.Errorf("can't find secret %s, reason: %v", key, err)
					err = conn.Close()
					if err != nil {
						klog.Errorf("close conn err: %v", err)
					}
					continue
				}
				certBytes, keyBytes, rootCA, err := getTLSCertAndKey(*s)
				if err != nil {
					klog.Errorf("get tls cert and tls key from k8s secret %s err: %v", key, err)
					err = conn.Close()
					if err != nil {
						klog.Errorf("close conn err: %v", err)
					}
					continue
				}
				certificate, err := tls.X509KeyPair(certBytes, keyBytes)
				if err != nil {
					klog.Errorf("transform x509 cert for tls server err: %v", err)
					err = conn.Close()
					if err != nil {
						klog.Errorf("close conn err: %v", err)
					}
					continue
				}
				tlsConfig := &tls.Config{
					Certificates: []tls.Certificate{certificate},
					CipherSuites: srv.options.CipherSuites,
					MinVersion:   srv.options.MinVersion,
					MaxVersion:   srv.options.MaxVersion,
				}
				if rootCA != nil {
					caCertPool := x509.NewCertPool()
					if ok := caCertPool.AppendCertsFromPEM(rootCA); !ok {
						klog.Errorf("ca cert invalid")
						err = conn.Close()
						if err != nil {
							klog.Errorf("close conn err: %v", err)
						}
						continue
					}
					tlsConfig.ClientCAs = caCertPool
					tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
				}
				conn = tls.Server(conn, tlsConfig)
			}
			proto, err := srv.newProto(conn)
			if err != nil {
				klog.Errorf("get proto from conn err: %v", err)
				err = conn.Close()
				if err != nil {
					klog.Errorf("close conn err: %v", err)
				}
				continue
			}
			srv.wg.Add(1)
			go func() {
				proto.Process()
				srv.wg.Done()
			}()
		}
	}
}

func (srv *Server) newProto(conn net.Conn) (protocol.Protocol, error) {
	// ingress traffic
	if srv.options.Exposed {
		// find all virtual services that bound to the gateway
		var vss []*istioapi.VirtualService
		vsList, err := controller.GetVirtualServiceLister().VirtualServices(srv.options.Namespace).List(labels.Everything())
		if err != nil {
			return nil, fmt.Errorf("list virtual service error %v", err)
		}
		for _, vs := range vsList {
			if vs.Spec.Gateways[0] == srv.options.GwName && reflect.DeepEqual(vs.Spec.Hosts, srv.options.Hosts) {
				vss = append(vss, vs)
			}
		}

		// get protocol
		if srv.options.Protocol == "HTTP" || srv.options.Protocol == "HTTPS" {
			for _, vs := range vss {
				proto := &http.HTTP{
					Conn:           conn,
					VirtualService: vs,
				}
				return proto, nil
			}
			return nil, fmt.Errorf("no match virtual service")
		} else if srv.options.Protocol == "TCP" {
			for _, vs := range vss {
				// TODO currently only one tcp route for a virtual service@Porunga
				if len(vs.Spec.Tcp) == 1 && len(vs.Spec.Tcp[0].Route) == 1 {
					proto := &tcp.TCP{
						Conn:         conn,
						SvcNamespace: srv.options.Namespace,
						SvcName:      vs.Spec.Tcp[0].Route[0].Destination.Host,
						Port:         int(vs.Spec.Tcp[0].Route[0].Destination.Port.Number),
					}
					return proto, nil
				}
			}
			return nil, fmt.Errorf("no match virtual service")
		} else {
			return nil, fmt.Errorf("protocol %s not supported", srv.options.Protocol)
		}
	}
	return nil, fmt.Errorf("egress traffic not supported")
}

// Stop stop
func (srv *Server) Stop() {
	close(srv.stop)
	err := srv.listener.Close()
	if err != nil {
		klog.Errorf("close Server err: %v", err)
	}
	srv.wg.Wait()
}

func getTLSCertAndKey(s v1.Secret) ([]byte, []byte, []byte, error) {
	if s.Type != "kubernetes.io/tls" {
		return nil, nil, nil, fmt.Errorf("secret %s not tls secret", s.Name)
	}
	if s.Data == nil {
		return nil, nil, nil, fmt.Errorf("secret %s data is empty", s.Name)
	}
	tlsCrt, ok := s.Data["tls.crt"]
	if !ok {
		return nil, nil, nil, fmt.Errorf("tls cert not found in secret %s data", s.Name)
	}
	tlsKey, ok := s.Data["tls.key"]
	if !ok {
		return nil, nil, nil, fmt.Errorf("tls key not found in secret %s data", s.Name)
	}
	rootCA, ok := s.Data["ca.crt"]
	if !ok {
		return tlsCrt, tlsKey, nil, nil
	}
	return tlsCrt, tlsKey, rootCA, nil
}
