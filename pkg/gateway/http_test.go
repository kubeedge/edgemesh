package gateway

import (
	"testing"

	networkingv1alpha3 "istio.io/api/networking/v1alpha3"
	"istio.io/client-go/pkg/apis/networking/v1alpha3"
)

func TestUriMatch(t *testing.T) {
	uriRules := []*networkingv1alpha3.StringMatch{
		{MatchType: &networkingv1alpha3.StringMatch_Prefix{Prefix: "/"}},
		{MatchType: &networkingv1alpha3.StringMatch_Prefix{Prefix: "/abc"}},
		{MatchType: &networkingv1alpha3.StringMatch_Prefix{Prefix: "/xyz"}},
		{MatchType: &networkingv1alpha3.StringMatch_Prefix{Prefix: "/hello*"}},
		{MatchType: &networkingv1alpha3.StringMatch_Prefix{Prefix: "/)(j3*a0"}},
	}
	reqUris := []string{
		"/",
		"/abc",
		"/abcd",
		"/xyz",
		"/hello-world",
	}
	for _, reqUri := range reqUris {
		for _, uriRule := range uriRules {
			t.Log(uriMatch(uriRule, reqUri))
		}
	}
}

func TestCheckHost(t *testing.T) {
	h := HTTP{
		VirtualService: &v1alpha3.VirtualService{
			Spec: networkingv1alpha3.VirtualService{
				Hosts: []string{"abc.com", "qwe.com"},
			},
		},
	}
	t.Log(h.checkHost(""))
	t.Log(h.checkHost("abc.com"))
	t.Log(h.checkHost("abc.com:80"))
	t.Log(h.checkHost("abc.com:9000"))
	t.Log(h.checkHost("abc.com#9000"))
	t.Log(h.checkHost("qwe.com:9000"))
	t.Log(h.checkHost("127.0.0.1:9000"))
	t.Log(h.checkHost("zxc.com"))
	t.Log(h.checkHost("zxc.com:9000"))
	t.Log(h.checkHost("jkl.io"))

	h = HTTP{
		VirtualService: &v1alpha3.VirtualService{
			Spec: networkingv1alpha3.VirtualService{
				Hosts: []string{"*"},
			},
		},
	}
	t.Log(h.checkHost(""))
	t.Log(h.checkHost("abc.com"))
	t.Log(h.checkHost("abc.com:80"))
	t.Log(h.checkHost("abc.com:9000"))
	t.Log(h.checkHost("abc.com#9000"))
	t.Log(h.checkHost("qwe.com:9000"))
	t.Log(h.checkHost("127.0.0.1:9000"))
	t.Log(h.checkHost("zxc.com"))
	t.Log(h.checkHost("zxc.com:9000"))
	t.Log(h.checkHost("jkl.io"))
}
