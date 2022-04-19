package userspace

import (
	"strings"
	"testing"

	"istio.io/api/meta/v1alpha1"
	"istio.io/api/networking/v1alpha3"
	istioapi "istio.io/client-go/pkg/apis/networking/v1alpha3"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/kubernetes/pkg/proxy"
)

func TestConsistentHashStrategy(t *testing.T) {
	lb := NewLoadBalancerEX()
	nodeName := "node1"
	const ns = "ns"
	const svc = "myservice"
	const portName1 = "mqtt"
	const portName2 = "http"
	lb.OnEndpointsAdd(&v1.Endpoints{
		TypeMeta: metav1.TypeMeta{Kind: "Endpoints"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      svc,
			Namespace: ns,
		},
		Subsets: []v1.EndpointSubset{{
			Addresses: []v1.EndpointAddress{{
				IP:       "192.168.0.1",
				Hostname: "pod1",
				NodeName: &nodeName,
				TargetRef: &v1.ObjectReference{
					Kind:      "Pod",
					Namespace: ns,
					Name:      "mypod-787756668b-6gczj",
				},
			},
				{
					IP:       "192.168.0.2",
					Hostname: "pod2",
					NodeName: &nodeName,
					TargetRef: &v1.ObjectReference{
						Kind:      "Pod",
						Namespace: ns,
						Name:      "mypod-787756668b-6gczi",
					},
				}},
			NotReadyAddresses: nil,
			Ports: []v1.EndpointPort{{
				Name:     portName1,
				Port:     1883,
				Protocol: "tcp",
			},
				{
					Name:     portName2,
					Port:     80,
					Protocol: "tcp",
				}},
		}},
	})

	namespacedName := types.NamespacedName{Namespace: ns, Name: svc}
	svcPortName1 := proxy.ServicePortName{NamespacedName: namespacedName, Port: portName1}
	svcPortName2 := proxy.ServicePortName{NamespacedName: namespacedName, Port: portName2}

	serviceCheck := func(svcPortName proxy.ServicePortName) {
		if state, ok := lb.services[svcPortName]; ok {
			if len(state.endpoints) != 2 {
				t.Errorf("length of endpoints of ServicePortName %v should be 2", svcPortName)
			}
		} else {
			t.Errorf("ServicePortName %v should exist", svcPortName)
		}
	}

	serviceCheck(svcPortName1)
	serviceCheck(svcPortName2)

	lb.OnDestinationRuleAdd(&istioapi.DestinationRule{
		TypeMeta:   metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: svc},
		Spec: v1alpha3.DestinationRule{
			TrafficPolicy: &v1alpha3.TrafficPolicy{
				LoadBalancer: &v1alpha3.LoadBalancerSettings{
					LbPolicy: &v1alpha3.LoadBalancerSettings_ConsistentHash{},
				},
			},
		},
		Status: v1alpha1.IstioStatus{},
	})

	strategyCheck := func(svcPortName proxy.ServicePortName, port string) {
		if _, ok := lb.strategyMap[svcPortName].(*ConsistentHashStrategy); !ok {
			t.Errorf("strategy of %v should be consistent hash", svcPortName1)
			return
		}

		ep, _, err := lb.NextEndpoint(
			svcPortName,
			nil,
			nil,
			false,
		)
		if err != nil {
			t.Error(err)
			return
		}
		parts := strings.Split(ep, ":")
		if len(parts) != 4 || parts[3] != port {
			t.Errorf("a wrong endpoint picked for ServicePortName %v", svcPortName)
		}
	}

	strategyCheck(svcPortName1, "1883")
	strategyCheck(svcPortName2, "80")
}
