/*
Copyright 2026 The KubeEdge Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package loadbalancer

import (
	"testing"

	networking "istio.io/api/networking/v1alpha3"
	istioapi "istio.io/client-go/pkg/apis/networking/v1alpha3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/kubernetes/pkg/proxy"
)

func TestDestinationRuleWithoutSupportedPolicy(t *testing.T) {
	cases := []struct {
		name    string
		traffic *networking.TrafficPolicy
	}{
		{"absent traffic policy", nil},
		{"TLS only", &networking.TrafficPolicy{Tls: &networking.ClientTLSSettings{Mode: networking.ClientTLSSettings_ISTIO_MUTUAL}}},
		{"empty LB", &networking.TrafficPolicy{LoadBalancer: &networking.LoadBalancerSettings{}}},
		{"unsupported LB", &networking.TrafficPolicy{LoadBalancer: &networking.LoadBalancerSettings{LbPolicy: &networking.LoadBalancerSettings_Simple{Simple: networking.LoadBalancerSettings_LEAST_CONN}}}},
	}
	for _, tc := range cases {
		for _, operation := range []string{"add", "update", "delete", "supported-to-unsupported"} {
			t.Run(tc.name+"/"+operation, func(t *testing.T) {
				port := proxy.ServicePortName{NamespacedName: types.NamespacedName{Namespace: "test", Name: "service"}, Port: "http"}
				lb := &LoadBalancer{services: map[proxy.ServicePortName]*balancerState{port: {}}, policyMap: map[proxy.ServicePortName]Policy{}}
				dr := &istioapi.DestinationRule{ObjectMeta: metav1.ObjectMeta{Name: "service", Namespace: "test"}, Spec: networking.DestinationRule{Host: "service", TrafficPolicy: tc.traffic}}
				oldDr := dr.DeepCopy()
				if operation == "supported-to-unsupported" {
					oldDr.Spec.TrafficPolicy = &networking.TrafficPolicy{LoadBalancer: &networking.LoadBalancerSettings{LbPolicy: &networking.LoadBalancerSettings_Simple{Simple: networking.LoadBalancerSettings_ROUND_ROBIN}}}
					lb.OnDestinationRuleAdd(oldDr)
				}
				if operation == "add" {
					lb.OnDestinationRuleAdd(dr)
				} else if operation == "delete" {
					lb.OnDestinationRuleDelete(dr)
				} else {
					lb.OnDestinationRuleUpdate(oldDr, dr)
				}
				if _, exists := lb.policyMap[port]; exists {
					t.Fatal("unsupported policy must fall back to the existing service balancer")
				}
			})
		}
	}
}

func TestDestinationRuleSupportedPolicy(t *testing.T) {
	port := proxy.ServicePortName{NamespacedName: types.NamespacedName{Namespace: "test", Name: "service"}, Port: "http"}
	lb := &LoadBalancer{services: map[proxy.ServicePortName]*balancerState{port: {}}, policyMap: map[proxy.ServicePortName]Policy{}}
	dr := &istioapi.DestinationRule{ObjectMeta: metav1.ObjectMeta{Name: "service", Namespace: "test"}, Spec: networking.DestinationRule{Host: "service", TrafficPolicy: &networking.TrafficPolicy{LoadBalancer: &networking.LoadBalancerSettings{LbPolicy: &networking.LoadBalancerSettings_Simple{Simple: networking.LoadBalancerSettings_ROUND_ROBIN}}}}}
	lb.OnDestinationRuleAdd(dr)
	lb.OnDestinationRuleUpdate(dr.DeepCopy(), dr)
	if p := lb.policyMap[port]; p == nil || p.Name() != RoundRobin {
		t.Fatal("supported policy lost")
	}
	lb.OnDestinationRuleDelete(dr)
	if _, exists := lb.policyMap[port]; exists {
		t.Fatal("deleted policy remains")
	}
}
