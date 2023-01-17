package gateway

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"reflect"
	"strings"
	"sync"

	istioapi "istio.io/client-go/pkg/apis/networking/v1alpha3"
	"k8s.io/klog/v2"

	"github.com/kubeedge/edgemesh/pkg/gateway/cache"
	"github.com/kubeedge/edgemesh/pkg/loadbalancer"
)

var (
	once                 sync.Once
	internalLoadBalancer *loadbalancer.LoadBalancer
)

func initLoadBalancer(loadbalancer *loadbalancer.LoadBalancer) {
	once.Do(func() {
		internalLoadBalancer = loadbalancer
	})
}

// Server is gateway server
type Server struct {
	listener *net.TCPListener
	stop     chan interface{}
	wg       sync.WaitGroup
	options  *ServerOptions
}

// ServerOptions options
type ServerOptions struct {
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

// NewServer new server and run
func NewServer(ip net.IP, port int, opts *ServerOptions) (*Server, error) {
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
		select {
		case _, isClosed := <-srv.stop:
			if isClosed {
				klog.Errorf("server stop to serve")
			}
			return
		default:
			conn, err := srv.listener.Accept()
			if err != nil {
				if !IsClosedError(err) {
					klog.Errorf("get conn error: %v", err)
				}
				continue
			}
			// tls
			if srv.options.CredentialName != "" {
				klog.Infof("tls required")
				key := cache.KeyFormat(srv.options.Namespace, srv.options.CredentialName)
				s, ok := cache.GetSecret(key)
				if !ok {
					klog.Errorf("can't find k8s secret %s in cache", key)
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
			// handle req
			go proto.Process()
		}
	}
}

func (srv *Server) newProto(conn net.Conn) (Protocol, error) {
	if srv.options.Exposed {
		// external traffic
		var vss []*istioapi.VirtualService
		// find all virtual services that bound to the gateway
		fn := func(key, value interface{}) bool {
			if vs, ok := value.(*istioapi.VirtualService); ok {
				if vs.Spec.Gateways[0] == srv.options.GwName &&
					reflect.DeepEqual(vs.Spec.Hosts, srv.options.Hosts) {
					vss = append(vss, vs)
				}
			}
			return true
		}
		cache.RangeVirtualServices(fn)
		// http
		if srv.options.Protocol == "HTTP" || srv.options.Protocol == "HTTPS" {
			for _, vs := range vss {
				proto := &HTTP{
					Conn:           conn,
					VirtualService: vs,
				}
				return proto, nil
			}
			return nil, fmt.Errorf("no match virtual service")
		} else if srv.options.Protocol == "TCP" {
			for _, vs := range vss {
				// currently only one tcp route for a virtual service
				if len(vs.Spec.Tcp) == 1 && len(vs.Spec.Tcp[0].Route) == 1 {
					proto := &TCP{
						Conn:         conn,
						SvcNamespace: srv.options.Namespace,
						SvcName:      vs.Spec.Tcp[0].Route[0].Destination.Host,
						SvcPort:      int(vs.Spec.Tcp[0].Route[0].Destination.Port.Number),
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
		klog.Errorf("close listener err: %v", err)
	}
	srv.wg.Wait()
}

func IsClosedError(err error) bool {
	return strings.HasPrefix(err.Error(), "use of closed network connection")
}
