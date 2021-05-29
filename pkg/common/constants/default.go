package constants

import v1 "k8s.io/api/core/v1"

// Resources
const (
	// Certificates
	DefaultConfigDir = "/etc/kubeedge/config/"

	// Config
	DefaultKubeContentType         = "application/vnd.kubernetes.protobuf"
	DefaultKubeConfig              = "/root/.kube/config"
	DefaultKubeNamespace           = v1.NamespaceAll
	DefaultKubeQPS                 = 100.0
	DefaultKubeBurst               = 200
	DefaultKubeUpdateNodeFrequency = 20

	// Controller
	DefaultServiceEventBuffer         = 1
	DefaultEndpointsEventBuffer       = 1
	DefaultDestinationRuleEventBuffer = 1
	DefaultGatewayEventBuffer         = 1

	// Resource
	ResourceTypeService     = "service"
	ResourceTypeEndpoints   = "endpoints"
	ResourceDestinationRule = "destinationRule"
	ResourceTypeGateway     = "gateway"
)
