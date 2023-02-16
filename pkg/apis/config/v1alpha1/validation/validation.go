package validation

import (
	"path/filepath"

	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/kubeedge/edgemesh/pkg/apis/config/defaults"
	"github.com/kubeedge/edgemesh/pkg/apis/config/v1alpha1"
	utilvalidation "github.com/kubeedge/kubeedge/pkg/util/validation"
)

func ValidateEdgeMeshAgentConfiguration(c *v1alpha1.EdgeMeshAgentConfig) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, ValidateKubeAPIConfig(c.KubeAPIConfig)...)
	allErrs = append(allErrs, ValidateModuleEdgeTunnel(c.Modules.EdgeTunnelConfig)...)
	allErrs = append(allErrs, ValidateModuleEdgeProxy(c.Modules.EdgeProxyConfig)...)
	return allErrs
}

func ValidateEdgeMeshGatewayConfiguration(c *v1alpha1.EdgeMeshGatewayConfig) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, ValidateKubeAPIConfig(c.KubeAPIConfig)...)
	allErrs = append(allErrs, ValidateModuleEdgeTunnel(c.Modules.EdgeTunnelConfig)...)
	return allErrs
}

func ValidateKubeAPIConfig(c *v1alpha1.KubeAPIConfig) field.ErrorList {
	allErrs := field.ErrorList{}
	if c.KubeConfig != "" && !filepath.IsAbs(c.KubeConfig) {
		allErrs = append(allErrs, field.Invalid(field.NewPath("kubeconfig"), c.KubeConfig, "kubeconfig need abs path"))
	}
	if c.KubeConfig != "" && !utilvalidation.FileIsExist(c.KubeConfig) {
		allErrs = append(allErrs, field.Invalid(field.NewPath("kubeconfig"), c.KubeConfig, "kubeconfig not exist"))
	}
	// TODO validate metaServerAddress
	return allErrs
}

func ValidateModuleEdgeProxy(c *v1alpha1.EdgeProxyConfig) field.ErrorList {
	if !c.Enable {
		return field.ErrorList{}
	}

	allErrs := field.ErrorList{}

	validServiceFilterModes := []defaults.ServiceFilterMode{"FilterIfLabelDoesNotExists", "FilterIfLabelExists"}
	if !isValidServiceFilterMode(c.ServiceFilterMode, validServiceFilterModes) {
		allErrs = append(allErrs, field.Invalid(field.NewPath("serviceFilterMode"), c.ServiceFilterMode, "invalid serviceFilterMode"))
	}

	return allErrs
}

func isValidServiceFilterMode(mode defaults.ServiceFilterMode, validValues []defaults.ServiceFilterMode) bool {
	for _, m := range validValues {
		if mode == m {
			return true
		}
	}
	return false
}

func ValidateModuleEdgeTunnel(c *v1alpha1.EdgeTunnelConfig) field.ErrorList {
	if !c.Enable {
		return field.ErrorList{}
	}

	allErrs := field.ErrorList{}
	validTransport := IsValidTransport(c.Transport)

	if len(validTransport) > 0 {
		for _, m := range validTransport {
			allErrs = append(allErrs, field.Invalid(field.NewPath("Transport"), c.Transport, m))
		}
	}

	return allErrs
}

func IsValidTransport(transport string) []string {
	var supportedTransports = []string{"tcp", "ws", "quic"}
	for _, tr := range supportedTransports {
		if transport == tr {
			return nil
		}
	}
	return supportedTransports
}
