package v1alpha1

import (
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/cloudcore/v1alpha1/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func ValidateEdgeMeshAgentConfiguration(c *EdgeMeshAgentConfig) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, validation.ValidateKubeAPIConfig(*c.KubeAPIConfig)...)
	allErrs = append(allErrs, ValidateModuleEdgeTunnel(*c.Modules.EdgeTunnelConfig)...)
	return allErrs
}

func ValidateEdgeMeshGatewayConfiguration(c *EdgeMeshGatewayConfig) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, validation.ValidateKubeAPIConfig(*c.KubeAPIConfig)...)
	allErrs = append(allErrs, ValidateModuleEdgeTunnel(*c.Modules.EdgeTunnelConfig)...)
	return allErrs
}

func ValidateModuleEdgeTunnel(c EdgeTunnelConfig) field.ErrorList {
	if !c.Enable {
		return field.ErrorList{}
	}

	allErrs := field.ErrorList{}
	validTransport := isValidTransport(c.Transport)

	if len(validTransport) > 0 {
		for _, m := range validTransport {
			allErrs = append(allErrs, field.Invalid(field.NewPath("Transport"), c.Transport, m))
		}
	}

	return allErrs
}

func isValidTransport(transport string) []string {
	var supportedTransports = []string{"tcp", "ws", "quic"}
	for _, tr := range supportedTransports {
		if transport == tr {
			return nil
		}
	}
	return supportedTransports
}
