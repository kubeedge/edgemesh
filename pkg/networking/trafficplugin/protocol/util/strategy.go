package util

import (
	"github.com/go-chassis/go-chassis/core/loadbalancer"
	apiv1alpha3 "istio.io/api/networking/v1alpha3"
	"k8s.io/klog/v2"

	"github.com/kubeedge/edgemesh/pkg/controller"
	"github.com/kubeedge/edgemesh/pkg/networking/trafficplugin/config"
	"github.com/kubeedge/edgemesh/pkg/networking/trafficplugin/loadbalancer/consistenthash"
)

// GetStrategyName returns load balance strategy name
func GetStrategyName(namespace, name string) string {
	var strategyName string
	// get default lb strategy from config file
	defaultStrategy := config.Config.LoadBalancer.DefaultLBStrategy
	// find destination rule bound to service
	dr, err := controller.GetDestinationRuleLister().DestinationRules(namespace).Get(name)
	if err != nil {
		klog.Warningf("failed to find %s.%s DestinationRule, use default strategy %s from config file", namespace, name, defaultStrategy)
		return defaultStrategy
	}

	switch lbPolicy := dr.Spec.TrafficPolicy.LoadBalancer.LbPolicy.(type) {
	case *apiv1alpha3.LoadBalancerSettings_Simple:
		strategyName = getSimpleLB(lbPolicy.Simple.String())
	case *apiv1alpha3.LoadBalancerSettings_ConsistentHash:
		strategyName = consistenthash.StrategyConsistentHash
	default:
		strategyName = defaultStrategy
	}
	klog.Infof("loadbalancer strategy: %s", strategyName)
	return strategyName
}

func getSimpleLB(simpleLb string) string {
	switch simpleLb {
	case "ROUND_ROBIN":
		simpleLb = loadbalancer.StrategyRoundRobin
	case "RANDOM":
		simpleLb = loadbalancer.StrategyRandom
	default:
		klog.Warningf("strategy not support %s, use default strategy: RoundRobin", simpleLb)
		simpleLb = config.Config.LoadBalancer.DefaultLBStrategy
	}
	return simpleLb
}
