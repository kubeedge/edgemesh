package networking

import (
	"fmt"

	apiv1alpha3 "istio.io/api/networking/v1alpha3"
	istioapi "istio.io/client-go/pkg/apis/networking/v1alpha3"
	v1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/edgemesh/pkg/common/constants"
	"github.com/kubeedge/edgemesh/pkg/controller"
	"github.com/kubeedge/edgemesh/pkg/networking/edgegateway"
	gatewayConfig "github.com/kubeedge/edgemesh/pkg/networking/edgegateway/config"
	discoveryConfig "github.com/kubeedge/edgemesh/pkg/networking/servicediscovery/config"
	"github.com/kubeedge/edgemesh/pkg/networking/servicediscovery/serviceproxy"
	"github.com/kubeedge/edgemesh/pkg/networking/trafficplugin/loadbalancer/consistenthash"
)

func handleServiceMessage(msg model.Message) {
	svc, ok := msg.GetContent().(*v1.Service)
	if !ok {
		klog.Warningf("object type: %T unsupported", svc)
		return
	}
	svcName := svc.Namespace + "." + svc.Name
	svcPorts := serviceproxy.GetSvcPorts(svc, svcName)
	clusterIP := svc.Spec.ClusterIP
	operation := msg.GetOperation()
	switch operation {
	case model.InsertOperation, model.UpdateOperation:
		serviceproxy.AddOrUpdateService(svcName, clusterIP, svcPorts)
	case model.DeleteOperation:
		serviceproxy.DeleteService(svcName, clusterIP)
	}
}

func handleEndpointsMessage(msg model.Message) {
	eps, ok := msg.GetContent().(*v1.Endpoints)
	if !ok {
		klog.Warningf("object type: %T unsupported", eps)
		return
	}
	if eps.Namespace == "kube-system" {
		return
	}
	operation := msg.GetOperation()
	switch operation {
	case model.InsertOperation:
	case model.UpdateOperation:
		// update hash ring if needed
		key := fmt.Sprintf("%s.%s", eps.Namespace, eps.Name)
		svc, err := controller.GetServiceLister().Services(eps.Namespace).Get(eps.Name)
		if err != nil || svc == nil {
			klog.V(4).Infof("no service %s exists", key)
			return
		}
		dr, err := controller.GetDestinationRuleLister().DestinationRules(eps.Namespace).Get(eps.Name)
		if err != nil || dr == nil {
			klog.V(4).Infof("no destinationRule %s exists", key)
			return
		}
		if isConsistentHashLB(dr) {
			klog.Infof("may need to update hash ring %s", key)
			consistenthash.UpdateHashRingByService(key, svc)
		}
	case model.DeleteOperation:
	}
}

func handleDestinationRuleMessage(msg model.Message) {
	dr, ok := msg.GetContent().(*istioapi.DestinationRule)
	if !ok {
		klog.Warningf("object type: %T unsupported", dr)
		return
	}
	key := fmt.Sprintf("%s.%s", dr.Namespace, dr.Name)
	operation := msg.GetOperation()
	switch operation {
	case model.InsertOperation:
		// when the load balance algorithm is consistent hash, we need to create a hash ring.
		if isConsistentHashLB(dr) {
			tryToCreateHashRing(dr.Namespace, dr.Name)
		}
	case model.UpdateOperation:
		// when the load balance algorithm is consistent hash, we need to create a hash ring.
		if isConsistentHashLB(dr) {
			tryToCreateHashRing(dr.Namespace, dr.Name)
		} else {
			// when the load balancing algorithm is updated, if it is no longer a consistent hash algorithm,
			// we need to delete the exists hash ring.
			if _, ok := consistenthash.GetHashRing(key); ok {
				consistenthash.DeleteHashRing(key)
			}
		}
	case model.DeleteOperation:
		if isConsistentHashLB(dr) {
			consistenthash.DeleteHashRing(key)
		}
	}
}

func tryToCreateHashRing(namespace, name string) {
	key := fmt.Sprintf("%s.%s", namespace, name)
	_, ok := consistenthash.GetHashRing(key)
	if ok {
		klog.Warningf("hash ring %s already exists in cache", key)
		return
	}
	dr, err := controller.GetDestinationRuleLister().DestinationRules(namespace).Get(name)
	if err != nil || dr == nil {
		klog.Errorf("failed to get the destinationRule bound to the service, reason: %v", err)
		return
	}
	if !isConsistentHashLB(dr) {
		klog.Warningf("not a consistent hash load balance strategy")
		return
	}
	svc, err := controller.GetServiceLister().Services(namespace).Get(name)
	if err != nil || svc == nil {
		klog.Errorf("failed to get the service bound to the destinationRule, reason: %v", err)
		return
	}
	// create new hash ring
	consistenthash.CreateHashRingByService(svc)
}

func isConsistentHashLB(dr *istioapi.DestinationRule) bool {
	var ok bool
	switch dr.Spec.TrafficPolicy.LoadBalancer.LbPolicy.(type) {
	case *apiv1alpha3.LoadBalancerSettings_ConsistentHash:
		ok = true
	default:
		ok = false
	}
	return ok
}

func handleGatewayMessage(msg model.Message) {
	gw, ok := msg.GetContent().(*istioapi.Gateway)
	if !ok {
		klog.Warningf("object type: %T unsupported", gw)
		return
	}
	operation := msg.GetOperation()
	gatewayManager := edgegateway.GetManager()
	switch operation {
	case model.InsertOperation:
		gatewayManager.AddGateway(gw)
	case model.UpdateOperation:
		gatewayManager.UpdateGateway(gw)
	case model.DeleteOperation:
		gatewayManager.DelGateway(gw)
	}
}

func process(msg model.Message) {
	resource := msg.GetResource()
	switch resource {
	case constants.ResourceTypeService:
		if discoveryConfig.Config.Enable {
			handleServiceMessage(msg)
		}
	case constants.ResourceTypeEndpoints:
		if discoveryConfig.Config.Enable || gatewayConfig.Config.Enable {
			handleEndpointsMessage(msg)
		}
	case constants.ResourceDestinationRule:
		if discoveryConfig.Config.Enable || gatewayConfig.Config.Enable {
			handleDestinationRuleMessage(msg)
		}
	case constants.ResourceTypeGateway:
		if gatewayConfig.Config.Enable {
			handleGatewayMessage(msg)
		}
	}
}
