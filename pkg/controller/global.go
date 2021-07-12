package controller

import (
	istioapi "istio.io/client-go/pkg/apis/networking/v1alpha3"
	istioinformers "istio.io/client-go/pkg/informers/externalversions"
	istiolisters "istio.io/client-go/pkg/listers/networking/v1alpha3"
	k8sapi "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/watch"
	k8sinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	k8slisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/klog/v2"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/edgemesh/pkg/common/client"
	"github.com/kubeedge/edgemesh/pkg/common/constants"
	"github.com/kubeedge/edgemesh/pkg/common/modules"
	"github.com/kubeedge/edgemesh/pkg/controller/manager"
)

var gc *GlobalController

type GlobalController struct {
	kubeClient kubernetes.Interface

	// lister
	podLister             k8slisters.PodLister
	secretLister          k8slisters.SecretLister
	serviceLister         k8slisters.ServiceLister
	endpointsLister       k8slisters.EndpointsLister
	destinationRuleLister istiolisters.DestinationRuleLister
	virtualServiceLister  istiolisters.VirtualServiceLister

	// manager
	serviceManager         *manager.ServiceManager
	endpointsManager       *manager.EndpointsManager
	destinationRuleManager *manager.DestinationRuleManager
	gatewayManager         *manager.GatewayManager
}

func (gc *GlobalController) syncService() {
	var operation string
	for {
		select {
		case <-beehiveContext.Done():
			klog.Warning("Stop global controller syncService loop")
			return
		case e := <-gc.serviceManager.Events():
			svc, ok := e.Object.(*k8sapi.Service)
			if !ok {
				klog.Warningf("Object type: %T unsupported", svc)
				continue
			}
			switch e.Type {
			case watch.Added:
				operation = model.InsertOperation
			case watch.Modified:
				operation = model.UpdateOperation
			case watch.Deleted:
				operation = model.DeleteOperation
			default:
				klog.Warningf("Service event type: %s unsupported", e.Type)
				continue
			}
			msg := model.NewMessage("")
			msg.BuildRouter(modules.NetworkingModuleName, modules.NetworkingGroupName, constants.ResourceTypeService, operation)
			msg.Content = svc
			klog.V(4).Infof("send msg: %+v", msg)
			beehiveContext.Send(modules.NetworkingModuleName, *msg)
		}
	}
}

func (gc *GlobalController) syncEndpoints() {
	var operation string
	for {
		select {
		case <-beehiveContext.Done():
			klog.Warning("Stop global controller syncEndpoints loop")
			return
		case e := <-gc.endpointsManager.Events():
			eps, ok := e.Object.(*k8sapi.Endpoints)
			if !ok {
				klog.Warningf("Object type: %T unsupported", eps)
				continue
			}
			switch e.Type {
			case watch.Added:
				operation = model.InsertOperation
			case watch.Modified:
				operation = model.UpdateOperation
			case watch.Deleted:
				operation = model.DeleteOperation
			default:
				klog.Warningf("Endpoints event type: %s unsupported", e.Type)
				continue
			}
			msg := model.NewMessage("")
			msg.BuildRouter(modules.NetworkingModuleName, modules.NetworkingGroupName, constants.ResourceTypeEndpoints, operation)
			msg.Content = eps
			klog.V(4).Infof("send msg: %+v", msg)
			beehiveContext.Send(modules.NetworkingModuleName, *msg)
		}
	}
}

func (gc *GlobalController) syncDestinationRule() {
	var operation string
	for {
		select {
		case <-beehiveContext.Done():
			klog.Warning("Stop global controller syncDestinationRule loop")
			return
		case e := <-gc.destinationRuleManager.Events():
			klog.V(4).Infof("Get destination rule events: event type: %s.", e.Type)
			dr, ok := e.Object.(*istioapi.DestinationRule)
			if !ok {
				klog.Warningf("object type: %T unsupported", dr)
				continue
			}
			switch e.Type {
			case watch.Added:
				operation = model.InsertOperation
			case watch.Modified:
				operation = model.UpdateOperation
			case watch.Deleted:
				operation = model.DeleteOperation
			default:
				klog.Warningf("DestinationRule event type: %s unsupported", e.Type)
				continue
			}
			msg := model.NewMessage("")
			msg.BuildRouter(modules.NetworkingModuleName, modules.NetworkingGroupName, constants.ResourceDestinationRule, operation)
			msg.Content = dr
			klog.V(4).Infof("send msg: %+v", msg)
			beehiveContext.Send(modules.NetworkingModuleName, *msg)
		}
	}
}

func (gc *GlobalController) syncGateway() {
	var operation string
	for {
		select {
		case <-beehiveContext.Done():
			klog.Warning("Stop global controller syncGateway loop")
			return
		case e := <-gc.gatewayManager.Events():
			klog.V(4).Infof("Get gateway events: event type: %s.", e.Type)
			gw, ok := e.Object.(*istioapi.Gateway)
			if !ok {
				klog.Warningf("object type: %T unsupported", gw)
				continue
			}
			switch e.Type {
			case watch.Added:
				operation = model.InsertOperation
			case watch.Modified:
				operation = model.UpdateOperation
			case watch.Deleted:
				operation = model.DeleteOperation
			default:
				klog.Warningf("Gateway event type: %s unsupported", e.Type)
				continue
			}
			msg := model.NewMessage("")
			msg.BuildRouter(modules.NetworkingModuleName, modules.NetworkingGroupName, constants.ResourceTypeGateway, operation)
			msg.Content = gw
			klog.V(4).Infof("send msg: %+v", msg)
			beehiveContext.Send(modules.NetworkingModuleName, *msg)
		}
	}
}

// Start GlobalController
func (gc *GlobalController) Start() error {
	klog.Info("start global controller")
	// service
	go gc.syncService()

	// endpoints
	go gc.syncEndpoints()

	// destination rule
	go gc.syncDestinationRule()

	// gateway
	go gc.syncGateway()
	return nil
}

func NewGlobalController(k8sInformerFactory k8sinformers.SharedInformerFactory,
	istioInformerFactory istioinformers.SharedInformerFactory) (*GlobalController, error) {
	// init informer
	podInformer := k8sInformerFactory.Core().V1().Pods()
	secretInformer := k8sInformerFactory.Core().V1().Secrets()
	svcInformer := k8sInformerFactory.Core().V1().Services()
	endpointsInformer := k8sInformerFactory.Core().V1().Endpoints()
	drInformer := istioInformerFactory.Networking().V1alpha3().DestinationRules()
	vsInformer := istioInformerFactory.Networking().V1alpha3().VirtualServices()
	gatewayInformer := istioInformerFactory.Networking().V1alpha3().Gateways()

	// init manager
	serviceManager, err := manager.NewServiceManager(svcInformer.Informer())
	if err != nil {
		klog.Warningf("Create service manager failed with error: %s", err)
		return nil, err
	}
	endpointsManager, err := manager.NewEndpointsManager(endpointsInformer.Informer())
	if err != nil {
		klog.Warningf("Create endpoints manager failed with error: %s", err)
		return nil, err
	}
	drManager, err := manager.NewDestinationRuleManager(drInformer.Informer())
	if err != nil {
		klog.Warningf("Create destinationRule manager failed with error: %s", err)
		return nil, err
	}
	gatewayManager, err := manager.NewGatewayManager(gatewayInformer.Informer())
	if err != nil {
		klog.Warningf("Create gateway manager failed with error: %s", err)
		return nil, err
	}

	gc = &GlobalController{
		kubeClient:             client.GetKubeClient(),
		podLister:              podInformer.Lister(),
		secretLister:           secretInformer.Lister(),
		serviceLister:          svcInformer.Lister(),
		endpointsLister:        endpointsInformer.Lister(),
		destinationRuleLister:  drInformer.Lister(),
		virtualServiceLister:   vsInformer.Lister(),
		serviceManager:         serviceManager,
		endpointsManager:       endpointsManager,
		destinationRuleManager: drManager,
		gatewayManager:         gatewayManager,
	}

	return gc, nil
}

func GetPodLister() k8slisters.PodLister {
	return gc.podLister
}

func GetSecretLister() k8slisters.SecretLister {
	return gc.secretLister
}

func GetServiceLister() k8slisters.ServiceLister {
	return gc.serviceLister
}

func GetEndPointsLister() k8slisters.EndpointsLister {
	return gc.endpointsLister
}

func GetDestinationRuleLister() istiolisters.DestinationRuleLister {
	return gc.destinationRuleLister
}

func GetVirtualServiceLister() istiolisters.VirtualServiceLister {
	return gc.virtualServiceLister
}
