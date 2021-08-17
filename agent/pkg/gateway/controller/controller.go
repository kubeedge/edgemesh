package controller

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"os"
	"reflect"
	"strings"
	"sync"

	apiv1alpha3 "istio.io/api/networking/v1alpha3"
	istioapi "istio.io/client-go/pkg/apis/networking/v1alpha3"
	istiolisters "istio.io/client-go/pkg/listers/networking/v1alpha3"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	k8slisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	"github.com/kubeedge/edgemesh/agent/pkg/chassis/protocol"
	"github.com/kubeedge/edgemesh/agent/pkg/chassis/protocol/tcp"
	"github.com/kubeedge/edgemesh/agent/pkg/gateway/config"
	"github.com/kubeedge/edgemesh/agent/pkg/gateway/util"
	"github.com/kubeedge/edgemesh/common/informers"
)

var (
	APIConn *GatewayController
	once    sync.Once
)

type GatewayController struct {
	secretLister k8slisters.SecretLister
	vsLister     istiolisters.VirtualServiceLister
	gwInformer   cache.SharedIndexInformer
	gwManager    *Manager
}

func Init(ifm *informers.Manager, cfg *config.EdgeGatewayConfig) {
	once.Do(func() {
		APIConn = &GatewayController{
			secretLister: ifm.GetKubeFactory().Core().V1().Secrets().Lister(),
			vsLister:     ifm.GetIstioFactory().Networking().V1alpha3().VirtualServices().Lister(),
			gwInformer:   ifm.GetIstioFactory().Networking().V1alpha3().Gateways().Informer(),
			gwManager:    NewGatewayManager(cfg),
		}
		ifm.RegisterInformer(APIConn.gwInformer)
		ifm.RegisterSyncedFunc(APIConn.onCacheSynced)
	})
}

func (c *GatewayController) onCacheSynced() {
	// set informers event handler
	c.gwInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.gwAdd, UpdateFunc: c.gwUpdate, DeleteFunc: c.gwDelete})
}

func (c *GatewayController) gwAdd(obj interface{}) {
	gw, ok := obj.(*istioapi.Gateway)
	if !ok {
		klog.Errorf("invalid type %v", obj)
		return
	}
	c.gwManager.AddGateway(gw)
}

func (c *GatewayController) gwUpdate(oldObj, newObj interface{}) {
	gw, ok := newObj.(*istioapi.Gateway)
	if !ok {
		klog.Errorf("invalid type %v", newObj)
		return
	}
	c.gwManager.UpdateGateway(gw)
}

func (c *GatewayController) gwDelete(obj interface{}) {
	gw, ok := obj.(*istioapi.Gateway)
	if !ok {
		klog.Errorf("invalid type %v", obj)
		return
	}
	c.gwManager.DeleteGateway(gw)
}

var (
	tlsVersionMaps = map[apiv1alpha3.ServerTLSSettings_TLSProtocol]uint16{
		apiv1alpha3.ServerTLSSettings_TLSV1_0: tls.VersionTLS10,
		apiv1alpha3.ServerTLSSettings_TLSV1_1: tls.VersionTLS11,
		apiv1alpha3.ServerTLSSettings_TLSV1_2: tls.VersionTLS12,
		apiv1alpha3.ServerTLSSettings_TLSV1_3: tls.VersionTLS13,
	}

	tlsCipherSuitesMaps = map[string]uint16{
		// TLS 1.0 - 1.2 cipher suites.
		"TLS_RSA_WITH_RC4_128_SHA":                      tls.TLS_RSA_WITH_RC4_128_SHA,
		"TLS_RSA_WITH_3DES_EDE_CBC_SHA":                 tls.TLS_RSA_WITH_3DES_EDE_CBC_SHA,
		"TLS_RSA_WITH_AES_128_CBC_SHA":                  tls.TLS_RSA_WITH_AES_128_CBC_SHA,
		"TLS_RSA_WITH_AES_256_CBC_SHA":                  tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		"TLS_RSA_WITH_AES_128_CBC_SHA256":               tls.TLS_RSA_WITH_AES_128_CBC_SHA256,
		"TLS_RSA_WITH_AES_128_GCM_SHA256":               tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
		"TLS_RSA_WITH_AES_256_GCM_SHA384":               tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
		"TLS_ECDHE_ECDSA_WITH_RC4_128_SHA":              tls.TLS_ECDHE_ECDSA_WITH_RC4_128_SHA,
		"TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA":          tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
		"TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA":          tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
		"TLS_ECDHE_RSA_WITH_RC4_128_SHA":                tls.TLS_ECDHE_RSA_WITH_RC4_128_SHA,
		"TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA":           tls.TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA,
		"TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA":            tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
		"TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA":            tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
		"TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256":       tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256,
		"TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256":         tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256,
		"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256":         tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		"TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256":       tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384":         tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		"TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384":       tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		"TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256":   tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
		"TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256": tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,

		// TLS 1.3 cipher suites.
		"TLS_AES_128_GCM_SHA256":       tls.TLS_AES_128_GCM_SHA256,
		"TLS_AES_256_GCM_SHA384":       tls.TLS_AES_256_GCM_SHA384,
		"TLS_CHACHA20_POLY1305_SHA256": tls.TLS_CHACHA20_POLY1305_SHA256,

		// Legacy names for the corresponding cipher suites with the correct _SHA256
		// suffix, retained for backward compatibility.
		"TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305":   tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
		"TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305": tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
	}
)

func transformTLSVersion(istioTLSVersion apiv1alpha3.ServerTLSSettings_TLSProtocol) uint16 {
	tlsVersion, ok := tlsVersionMaps[istioTLSVersion]
	if !ok {
		return tls.VersionTLS10
	}
	return tlsVersion
}

func transformTLSCipherSuites(istioCipherSuites []string) (cipherSuites []uint16) {
	for _, c := range istioCipherSuites {
		if cs, ok := tlsCipherSuitesMaps[c]; ok {
			cipherSuites = append(cipherSuites, cs)
		}
	}
	return
}

// Manager is gateway manager
type Manager struct {
	ipArray          []net.IP
	lock             sync.Mutex
	serversByGateway map[string][]*Server // gatewayNamespace.gatewayName --> servers
}

func NewGatewayManager(c *config.EdgeGatewayConfig) *Manager {
	mgr := &Manager{
		serversByGateway: make(map[string][]*Server),
	}
	klog.V(4).Infof("start get ips which need listen...")
	var err error
	mgr.ipArray, err = util.GetIPsNeedListen(c)
	if err != nil {
		klog.Fatalf("get GetIPsNeedListen err: %v", err)
	}
	klog.V(4).Infof("listen ips: %+v", mgr.ipArray)
	return mgr
}

// AddGateway add a gateway server
func (mgr *Manager) AddGateway(gw *istioapi.Gateway) {
	if gw == nil {
		klog.Errorf("gateway is nil")
		return
	}
	key := fmt.Sprintf("%s.%s", gw.Namespace, gw.Name)
	var gatewayServers []*Server
	for _, ip := range mgr.ipArray {
		for _, s := range gw.Spec.Servers {
			opts := &ServerOptions{
				Exposed:   true,
				GwName:    gw.Name,
				Namespace: gw.Namespace,
				Hosts:     s.Hosts,
				Protocol:  s.Port.Protocol,
			}
			if s.Tls != nil && s.Tls.CredentialName != "" {
				opts.CredentialName = s.Tls.CredentialName
				opts.MinVersion = transformTLSVersion(s.Tls.MinProtocolVersion)
				opts.MaxVersion = transformTLSVersion(s.Tls.MaxProtocolVersion)
				opts.CipherSuites = transformTLSCipherSuites(s.Tls.CipherSuites)
			}
			gatewayServer, err := NewServer(ip, int(s.Port.Number), opts)
			if err != nil {
				klog.Warningf("new gateway server on port %d error: %v", int(s.Port.Number), err)
				if strings.Contains(err.Error(), "address already in use") {
					klog.Errorf("new gateway server on port %d error: %v. please wait, maybe old pod is deleting.", int(s.Port.Number), err)
				}
				continue
			}
			gatewayServers = append(gatewayServers, gatewayServer)
		}
	}

	mgr.lock.Lock()
	mgr.serversByGateway[key] = gatewayServers
	mgr.lock.Unlock()
}

// UpdateGateway update a gateway server
func (mgr *Manager) UpdateGateway(gw *istioapi.Gateway) {
	mgr.lock.Lock()
	defer mgr.lock.Unlock()

	if gw == nil {
		klog.Errorf("gateway is nil")
		return
	}
	// shutdown old servers
	key := fmt.Sprintf("%s.%s", gw.Namespace, gw.Name)
	if oldGatewayServers, ok := mgr.serversByGateway[key]; ok {
		for _, gatewayServer := range oldGatewayServers {
			// block
			gatewayServer.Stop()
		}
	}
	delete(mgr.serversByGateway, key)

	// start new servers
	var newGatewayServers []*Server
	for _, ip := range mgr.ipArray {
		for _, s := range gw.Spec.Servers {
			opts := &ServerOptions{
				Exposed:   true,
				GwName:    gw.Name,
				Namespace: gw.Namespace,
				Hosts:     s.Hosts,
				Protocol:  s.Port.Protocol,
			}
			if s.Tls != nil && s.Tls.CredentialName != "" {
				opts.CredentialName = s.Tls.CredentialName
				opts.MinVersion = transformTLSVersion(s.Tls.MinProtocolVersion)
				opts.MaxVersion = transformTLSVersion(s.Tls.MaxProtocolVersion)
				opts.CipherSuites = transformTLSCipherSuites(s.Tls.CipherSuites)
			}
			gatewayServer, err := NewServer(ip, int(s.Port.Number), opts)
			if err != nil {
				klog.Warningf("new gateway server on port %d error: %v", int(s.Port.Number), err)
				if strings.Contains(err.Error(), "address already in use") {
					klog.Errorf("new gateway server on port %d error: %v. please wait, maybe old pod is deleting.", int(s.Port.Number), err)
					os.Exit(1)
				}
				continue
			}
			newGatewayServers = append(newGatewayServers, gatewayServer)
		}
	}
	mgr.serversByGateway[key] = newGatewayServers
}

// DeleteGateway delete a gateway server
func (mgr *Manager) DeleteGateway(gw *istioapi.Gateway) {
	mgr.lock.Lock()
	defer mgr.lock.Unlock()

	key := fmt.Sprintf("%s.%s", gw.Namespace, gw.Name)
	gatewayServers, ok := mgr.serversByGateway[key]
	if !ok {
		klog.Warningf("delete gateway %s with no servers", key)
		return
	}
	for _, gatewayServer := range gatewayServers {
		// block
		gatewayServer.Stop()
	}
	delete(mgr.serversByGateway, key)
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
				s, err := APIConn.secretLister.Secrets(srv.options.Namespace).Get(srv.options.CredentialName)
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
		vsList, err := APIConn.vsLister.VirtualServices(srv.options.Namespace).List(labels.Everything())
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
				// TODO: switch to http process when tunnel http proxy implemented
				proto := &tcp.TCP{
					Conn:         conn,
					SvcNamespace: srv.options.Namespace,
					SvcName:      vs.Spec.Http[0].Route[0].Destination.Host,
					Port:         int(vs.Spec.Http[0].Route[0].Destination.Port.Number),
				}
				return proto, nil
			}
			return nil, fmt.Errorf("no match virtual service")
		} else if srv.options.Protocol == "TCP" {
			for _, vs := range vss {
				// TODO: currently only one tcp route for a virtual service@Porunga
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
