package edgegateway

import (
	"fmt"
	"net"
	"os"
	"strings"
	"sync"

	istioapi "istio.io/client-go/pkg/apis/networking/v1alpha3"
	"k8s.io/klog/v2"

	"github.com/kubeedge/edgemesh/pkg/networking/edgegateway/server"
	"github.com/kubeedge/edgemesh/pkg/networking/edgegateway/util"
)

var (
	gatewayManager *Manager
	ipArray        []net.IP
	once           sync.Once
)

// Manager is gateway manager
type Manager struct {
	lock             sync.RWMutex
	serversByGateway map[string][]*server.Server // gatewayNamespace.gatewayName --> servers
}

func Init() {
	once.Do(func() {
		gatewayManager = &Manager{}
		gatewayManager.serversByGateway = make(map[string][]*server.Server)
		klog.Infof("start get ips which need listen...")
		var err error
		ipArray, err = util.GetIPsNeedListen()
		if err != nil {
			klog.Errorf("get GetIPsNeedListen err: %v", err)
			return
		}
		klog.Infof("ipArray is %+v", ipArray)
	})
}

// Add add
func (gm *Manager) AddGateway(gw *istioapi.Gateway) {
	gm.lock.Lock()
	defer gm.lock.Unlock()

	if gw == nil {
		klog.Errorf("gateway is nil")
		return
	}
	key := fmt.Sprintf("%s.%s", gw.Namespace, gw.Name)
	var gatewayServers []*server.Server
	for _, ip := range ipArray {
		for _, s := range gw.Spec.Servers {
			opts := &server.Options{
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
			gatewayServer, err := server.NewServer(ip, int(s.Port.Number), opts)
			if err != nil {
				klog.Warningf("new gateway server on port %d error: %v", int(s.Port.Number), err)
				if strings.Contains(err.Error(), "address already in use") {
					klog.Errorf("new gateway server on port %d error: %v. please wait, maybe old pod is deleting.", int(s.Port.Number), err)
					os.Exit(1)
				}
				continue
			}
			gatewayServers = append(gatewayServers, gatewayServer)
		}
	}
	gm.serversByGateway[key] = gatewayServers
}

// Update update
func (gm *Manager) UpdateGateway(gw *istioapi.Gateway) {
	gm.lock.Lock()
	defer gm.lock.Unlock()

	if gw == nil {
		klog.Errorf("gateway is nil")
		return
	}
	// shutdown old servers
	key := fmt.Sprintf("%s.%s", gw.Namespace, gw.Name)
	if oldGatewayServers, ok := gm.serversByGateway[key]; ok {
		for _, gatewayServer := range oldGatewayServers {
			// block
			gatewayServer.Stop()
		}
	}
	delete(gm.serversByGateway, key)

	// start new servers
	var newGatewayServers []*server.Server
	for _, ip := range ipArray {
		for _, s := range gw.Spec.Servers {
			opts := &server.Options{
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
			gatewayServer, err := server.NewServer(ip, int(s.Port.Number), opts)
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
	gm.serversByGateway[key] = newGatewayServers
}

// Del del
func (gm *Manager) DelGateway(gw *istioapi.Gateway) {
	gm.lock.Lock()
	defer gm.lock.Unlock()

	key := fmt.Sprintf("%s.%s", gw.Namespace, gw.Name)
	gatewayServers, ok := gm.serversByGateway[key]
	if !ok {
		klog.Warningf("delete gateway %s with no servers", key)
		return
	}
	for _, gatewayServer := range gatewayServers {
		// block
		gatewayServer.Stop()
	}
	delete(gm.serversByGateway, key)
}

// GetManager get manager
func GetManager() *Manager {
	return gatewayManager
}
