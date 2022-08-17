package validation

import (
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/kubeedge/edgemesh/agent/cmd/edgemesh-agent/app/config"
	tunnelconfig "github.com/kubeedge/edgemesh/agent/pkg/tunnel/config"
	utilvalidation "github.com/kubeedge/edgemesh/common/util/validation"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/cloudcore/v1alpha1/validation"
)

func ValidateEdgeMeshAgentConfiguration(c *config.EdgeMeshAgentConfig) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, validation.ValidateKubeAPIConfig(*c.KubeAPIConfig)...)
	allErrs = append(allErrs, ValidateModuleTunnelAgent(*c.Modules.EdgeTunnelConfig)...)
	return allErrs
}

func ValidateModuleTunnelAgent(c tunnelconfig.EdgeTunnelConfig) field.ErrorList {
	if !c.Enable {
		return field.ErrorList{}
	}

	allErrs := field.ErrorList{}
	validTransport := utilvalidation.IsValidTransport(c.Transport)

	if len(validTransport) > 0 {
		for _, m := range validTransport {
			allErrs = append(allErrs, field.Invalid(field.NewPath("Transport"), c.Transport, m))
		}
	}

	return allErrs
}
