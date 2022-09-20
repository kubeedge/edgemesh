package gateway

import (
	"testing"

	istiov1alpha3 "istio.io/api/networking/v1alpha3"
)

func TestUriMatch(t *testing.T) {
	uriRules := []*istiov1alpha3.StringMatch{
		{MatchType: &istiov1alpha3.StringMatch_Prefix{Prefix: "/"}},
		{MatchType: &istiov1alpha3.StringMatch_Prefix{Prefix: "/abc"}},
		{MatchType: &istiov1alpha3.StringMatch_Prefix{Prefix: "/xyz"}},
		{MatchType: &istiov1alpha3.StringMatch_Prefix{Prefix: "/hello*"}},
		{MatchType: &istiov1alpha3.StringMatch_Prefix{Prefix: "/)(j3*a0"}},
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
