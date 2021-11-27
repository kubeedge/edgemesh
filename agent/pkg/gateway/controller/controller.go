package controller

import (
	"sync"

	istiolisters "istio.io/client-go/pkg/listers/networking/v1alpha3"
	k8slisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	"github.com/kubeedge/edgemesh/agent/pkg/gateway/config"
	"github.com/kubeedge/edgemesh/common/informers"
)

var (
	APIConn *GatewayController
	once    sync.Once
)

type GatewayController struct {
	secretLister k8slisters.SecretLister
	vsLister     istiolisters.VirtualServiceLister

	sync.RWMutex
	gwInformer      cache.SharedIndexInformer
	gwEventHandlers map[string]cache.ResourceEventHandlerFuncs // key: gateway event handler name
}

func Init(ifm *informers.Manager, cfg *config.EdgeGatewayConfig) {
	once.Do(func() {
		APIConn = &GatewayController{
			secretLister:    ifm.GetKubeFactory().Core().V1().Secrets().Lister(),
			vsLister:        ifm.GetIstioFactory().Networking().V1alpha3().VirtualServices().Lister(),
			gwInformer:      ifm.GetIstioFactory().Networking().V1alpha3().Gateways().Informer(),
			gwEventHandlers: make(map[string]cache.ResourceEventHandlerFuncs),
		}
		ifm.RegisterInformer(APIConn.gwInformer)
		ifm.RegisterSyncedFunc(APIConn.onCacheSynced)
	})
}

func (c *GatewayController) onCacheSynced() {
	for name, funcs := range c.gwEventHandlers {
		klog.V(4).Infof("enable gateway event handler funcs: %s", name)
		c.gwInformer.AddEventHandler(funcs)
	}
}

func (c *GatewayController) SetGatewayEventHandlers(name string, handlerFuncs cache.ResourceEventHandlerFuncs) {
	c.Lock()
	if _, exist := c.gwEventHandlers[name]; exist {
		klog.Warningf("gateway event handler %s already exists, it will be overwritten!", name)
	}
	c.gwEventHandlers[name] = handlerFuncs
	c.Unlock()
}

func (c *GatewayController) GetSecretLister() k8slisters.SecretLister {
	return c.secretLister
}

func (c *GatewayController) GetVsLister() istiolisters.VirtualServiceLister {
	return c.vsLister
}
