package manager

import (
	"fmt"
	"net"
	"os"
	"strings"
	"sync"

	istioapi "istio.io/client-go/pkg/apis/networking/v1alpha3"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	"github.com/kubeedge/edgemesh/agent/pkg/gateway/config"
	"github.com/kubeedge/edgemesh/agent/pkg/gateway/controller"
	"github.com/kubeedge/edgemesh/agent/pkg/gateway/util"
)

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
	klog.Infof("gateway listen ips: %+v", mgr.ipArray)
	// set gateway-manager event handler funcs
	controller.APIConn.SetGatewayEventHandlers("gateway-manager", cache.ResourceEventHandlerFuncs{
		AddFunc: mgr.gwAdd, UpdateFunc: mgr.gwUpdate, DeleteFunc: mgr.gwDelete})
	return mgr
}

func (mgr *Manager) gwAdd(obj interface{}) {
	gw, ok := obj.(*istioapi.Gateway)
	if !ok {
		klog.Errorf("invalid type %v", obj)
		return
	}
	mgr.addGateway(gw)
}

func (mgr *Manager) gwUpdate(oldObj, newObj interface{}) {
	gw, ok := newObj.(*istioapi.Gateway)
	if !ok {
		klog.Errorf("invalid type %v", newObj)
		return
	}
	mgr.updateGateway(gw)
}

func (mgr *Manager) gwDelete(obj interface{}) {
	gw, ok := obj.(*istioapi.Gateway)
	if !ok {
		klog.Errorf("invalid type %v", obj)
		return
	}
	mgr.deleteGateway(gw)
}

// addGateway add a gateway server
func (mgr *Manager) addGateway(gw *istioapi.Gateway) {
	mgr.lock.Lock()
	defer mgr.lock.Unlock()

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
			klog.Infof("gateway `%s` add server on %s:%d", key, ip.String(), s.Port.Number)
		}
	}

	mgr.serversByGateway[key] = gatewayServers
}

// updateGateway update a gateway server
func (mgr *Manager) updateGateway(gw *istioapi.Gateway) {
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
			klog.Infof("gateway `%s` update server on %s:%d", key, ip.String(), s.Port.Number)
		}
	}
	mgr.serversByGateway[key] = newGatewayServers
}

// deleteGateway delete a gateway server
func (mgr *Manager) deleteGateway(gw *istioapi.Gateway) {
	mgr.lock.Lock()
	defer mgr.lock.Unlock()

	key := fmt.Sprintf("%s.%s", gw.Namespace, gw.Name)
	klog.Infof("delete gateway %s", key)
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
