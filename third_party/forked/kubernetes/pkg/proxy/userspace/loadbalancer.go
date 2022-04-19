// This package is copied from Kubernetes project.
// https://github.com/kubernetes/kubernetes/blob/v1.23.0/pkg/proxy/userspace/loadbalancer.go
// Added DestinationRuleHandler to listen to the events of the destination rule object
package userspace

import (
	"net"
	"net/http"

	"k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/proxy"

	proxyconfig "github.com/kubeedge/edgemesh/third_party/forked/kubernetes/pkg/proxy/config"
)

// LoadBalancer is an interface for distributing incoming requests to service endpoints.
type LoadBalancer interface {
	// NextEndpoint returns the endpoint to handle a request for the given
	// service-port and source address.
	NextEndpoint(service proxy.ServicePortName, srcAddr net.Addr, netConn net.Conn, sessionAffinityReset bool) (string, *http.Request, error)
	NewService(service proxy.ServicePortName, sessionAffinityType v1.ServiceAffinity, stickyMaxAgeSeconds int) error
	DeleteService(service proxy.ServicePortName)
	CleanupStaleStickySessions(service proxy.ServicePortName)
	ServiceHasEndpoints(service proxy.ServicePortName) bool

	proxyconfig.EndpointsHandler
	proxyconfig.DestinationRuleHandler
}
