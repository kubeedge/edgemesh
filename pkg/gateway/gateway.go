package gateway

import (
	"fmt"
	"os"
	"strings"

	istiv1alpha3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	istioinformers "istio.io/client-go/pkg/informers/externalversions"
	v1 "k8s.io/api/core/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/informers"
	toolscache "k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	"github.com/kubeedge/edgemesh/pkg/apis/config/defaults"
	"github.com/kubeedge/edgemesh/pkg/gateway/cache"
)

func (gw *EdgeGateway) Run() error {
	kubeInformerFactory := informers.NewSharedInformerFactory(gw.kubeClient, gw.syncPeriod)
	secretInformer := kubeInformerFactory.Core().V1().Secrets()
	secretInformer.Informer().AddEventHandlerWithResyncPeriod(
		toolscache.ResourceEventHandlerFuncs{
			AddFunc:    gw.handleAddSecret,
			UpdateFunc: gw.handleUpdateSecret,
			DeleteFunc: gw.handleDeleteSecret,
		},
		gw.syncPeriod,
	)
	go gw.runSecret(secretInformer.Informer().HasSynced, gw.stopCh)

	istioInformerFactory := istioinformers.NewSharedInformerFactory(gw.istioClient, gw.syncPeriod)
	gwInformer := istioInformerFactory.Networking().V1alpha3().Gateways()
	gwInformer.Informer().AddEventHandlerWithResyncPeriod(
		toolscache.ResourceEventHandlerFuncs{
			AddFunc:    gw.handleAddGateway,
			UpdateFunc: gw.handleUpdateGateway,
			DeleteFunc: gw.handleDeleteGateway,
		},
		gw.syncPeriod,
	)
	go gw.runGateway(gwInformer.Informer().HasSynced, gw.stopCh)
	vsInformer := istioInformerFactory.Networking().V1alpha3().VirtualServices()
	vsInformer.Informer().AddEventHandlerWithResyncPeriod(
		toolscache.ResourceEventHandlerFuncs{
			AddFunc:    gw.handleAddVirtualService,
			UpdateFunc: gw.handleUpdateVirtualService,
			DeleteFunc: gw.handleDeleteVirtualService,
		},
		gw.syncPeriod,
	)
	go gw.runVirtualService(vsInformer.Informer().HasSynced, gw.stopCh)

	kubeInformerFactory.Start(gw.stopCh)
	istioInformerFactory.Start(gw.stopCh)

	gw.loadBalancer.Config.Caller = defaults.GatewayCaller
	err := gw.loadBalancer.Run()
	if err != nil {
		return fmt.Errorf("failed to run loadBalancer: %w", err)
	}

	return nil
}

// OnGatewayAdd add a gateway server
func (gw *EdgeGateway) OnGatewayAdd(gateway *istiv1alpha3.Gateway) {
	gw.lock.Lock()
	defer gw.lock.Unlock()

	if gw == nil {
		klog.Errorf("gateway is nil")
		return
	}

	key := fmt.Sprintf("%s.%s", gateway.Namespace, gateway.Name)
	var gatewayServers []*Server
	for _, ip := range gw.listenIPs {
		for _, s := range gateway.Spec.Servers {
			opts := &ServerOptions{
				Exposed:   true,
				GwName:    gateway.Name,
				Namespace: gateway.Namespace,
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

	gw.serversByGateway[key] = gatewayServers
}

// OnGatewayUpdate update a gateway server
func (gw *EdgeGateway) OnGatewayUpdate(oldGateway, gateway *istiv1alpha3.Gateway) {
	gw.lock.Lock()
	defer gw.lock.Unlock()

	if gw == nil {
		klog.Errorf("gateway is nil")
		return
	}

	// shutdown old servers
	key := fmt.Sprintf("%s.%s", gateway.Namespace, gateway.Name)
	if oldGatewayServers, ok := gw.serversByGateway[key]; ok {
		for _, gatewayServer := range oldGatewayServers {
			// block
			gatewayServer.Stop()
		}
	}
	delete(gw.serversByGateway, key)

	// start new servers
	var newGatewayServers []*Server
	for _, ip := range gw.listenIPs {
		for _, s := range gateway.Spec.Servers {
			opts := &ServerOptions{
				Exposed:   true,
				GwName:    gateway.Name,
				Namespace: gateway.Namespace,
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
	gw.serversByGateway[key] = newGatewayServers
}

// OnGatewayDelete delete a gateway server
func (gw *EdgeGateway) OnGatewayDelete(gateway *istiv1alpha3.Gateway) {
	gw.lock.Lock()
	defer gw.lock.Unlock()

	key := fmt.Sprintf("%s.%s", gateway.Namespace, gateway.Name)
	klog.Infof("delete gateway %s", key)
	gatewayServers, ok := gw.serversByGateway[key]
	if !ok {
		klog.Warningf("delete gateway %s with no servers", key)
		return
	}
	for _, gatewayServer := range gatewayServers {
		// block
		gatewayServer.Stop()
	}
	delete(gw.serversByGateway, key)
}

func (gw *EdgeGateway) runGateway(listerSynced toolscache.InformerSynced, stopCh <-chan struct{}) {
	klog.InfoS("Starting EdgeGateway gateway controller")

	if !toolscache.WaitForNamedCacheSync("EdgeGateway gateway", stopCh, listerSynced) {
		return
	}
}

func (gw *EdgeGateway) handleAddGateway(obj interface{}) {
	gateway, ok := obj.(*istiv1alpha3.Gateway)
	if !ok {
		utilruntime.HandleError(fmt.Errorf("unexpected object type: %v", obj))
		return
	}
	gw.OnGatewayAdd(gateway)
}

func (gw *EdgeGateway) handleUpdateGateway(oldObj, newObj interface{}) {
	oldGateway, ok := oldObj.(*istiv1alpha3.Gateway)
	if !ok {
		utilruntime.HandleError(fmt.Errorf("unexpected object type: %v", oldObj))
		return
	}
	gateway, ok := newObj.(*istiv1alpha3.Gateway)
	if !ok {
		utilruntime.HandleError(fmt.Errorf("unexpected object type: %v", newObj))
		return
	}
	gw.OnGatewayUpdate(oldGateway, gateway)
}

func (gw *EdgeGateway) handleDeleteGateway(obj interface{}) {
	gateway, ok := obj.(*istiv1alpha3.Gateway)
	if !ok {
		tombstone, ok := obj.(toolscache.DeletedFinalStateUnknown)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("unexpected object type: %v", obj))
			return
		}
		if gateway, ok = tombstone.Obj.(*istiv1alpha3.Gateway); !ok {
			utilruntime.HandleError(fmt.Errorf("unexpected object type: %v", obj))
			return
		}
	}
	gw.OnGatewayDelete(gateway)
}

func (gw *EdgeGateway) runSecret(listerSynced toolscache.InformerSynced, stopCh <-chan struct{}) {
	klog.InfoS("Starting EdgeGateway secret controller")

	if !toolscache.WaitForNamedCacheSync("EdgeGateway secret", stopCh, listerSynced) {
		return
	}
}

func (gw *EdgeGateway) handleAddSecret(obj interface{}) {
	secret, ok := obj.(*v1.Secret)
	if !ok {
		utilruntime.HandleError(fmt.Errorf("unexpected object type: %v", obj))
		return
	}
	key := cache.KeyFormat(secret.Namespace, secret.Name)
	cache.UpdateSecret(key, secret)
}

func (gw *EdgeGateway) handleUpdateSecret(oldObj, newObj interface{}) {
	oldSecret, ok := oldObj.(*v1.Secret)
	if !ok {
		utilruntime.HandleError(fmt.Errorf("unexpected object type: %v", oldObj))
		return
	}
	secret, ok := newObj.(*v1.Secret)
	if !ok {
		utilruntime.HandleError(fmt.Errorf("unexpected object type: %v", newObj))
		return
	}
	oldKey := cache.KeyFormat(oldSecret.Namespace, oldSecret.Name)
	cache.DeleteSecret(oldKey)
	key := cache.KeyFormat(secret.Namespace, secret.Name)
	cache.UpdateSecret(key, secret)
}

func (gw *EdgeGateway) handleDeleteSecret(obj interface{}) {
	secret, ok := obj.(*v1.Secret)
	if !ok {
		tombstone, ok := obj.(toolscache.DeletedFinalStateUnknown)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("unexpected object type: %v", obj))
			return
		}
		if secret, ok = tombstone.Obj.(*v1.Secret); !ok {
			utilruntime.HandleError(fmt.Errorf("unexpected object type: %v", obj))
			return
		}
	}
	key := cache.KeyFormat(secret.Namespace, secret.Name)
	cache.DeleteSecret(key)
}

func (gw *EdgeGateway) runVirtualService(listerSynced toolscache.InformerSynced, stopCh <-chan struct{}) {
	klog.InfoS("Starting EdgeGateway virtualService controller")

	if !toolscache.WaitForNamedCacheSync("EdgeGateway virtualService", stopCh, listerSynced) {
		return
	}
}

func (gw *EdgeGateway) handleAddVirtualService(obj interface{}) {
	vs, ok := obj.(*istiv1alpha3.VirtualService)
	if !ok {
		utilruntime.HandleError(fmt.Errorf("unexpected object type: %v", obj))
		return
	}
	key := cache.KeyFormat(vs.Namespace, vs.Name)
	cache.UpdateVirtualService(key, vs)
}

func (gw *EdgeGateway) handleUpdateVirtualService(oldObj, newObj interface{}) {
	oldVs, ok := oldObj.(*istiv1alpha3.VirtualService)
	if !ok {
		utilruntime.HandleError(fmt.Errorf("unexpected object type: %v", oldObj))
		return
	}
	vs, ok := newObj.(*istiv1alpha3.VirtualService)
	if !ok {
		utilruntime.HandleError(fmt.Errorf("unexpected object type: %v", newObj))
		return
	}
	oldKey := cache.KeyFormat(oldVs.Namespace, oldVs.Name)
	cache.DeleteVirtualService(oldKey)
	key := cache.KeyFormat(vs.Namespace, vs.Name)
	cache.UpdateVirtualService(key, vs)
}

func (gw *EdgeGateway) handleDeleteVirtualService(obj interface{}) {
	vs, ok := obj.(*istiv1alpha3.VirtualService)
	if !ok {
		tombstone, ok := obj.(toolscache.DeletedFinalStateUnknown)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("unexpected object type: %v", obj))
			return
		}
		if vs, ok = tombstone.Obj.(*istiv1alpha3.VirtualService); !ok {
			utilruntime.HandleError(fmt.Errorf("unexpected object type: %v", obj))
			return
		}
	}
	key := cache.KeyFormat(vs.Namespace, vs.Name)
	cache.DeleteVirtualService(key)
}
