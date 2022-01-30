package validation

import (
	"k8s.io/apimachinery/pkg/util/validation/field"

	utilvalidation "github.com/kubeedge/edgemesh/common/util/validation"
	"github.com/kubeedge/edgemesh/server/cmd/edgemesh-server/app/config"
	tunnelconfig "github.com/kubeedge/edgemesh/server/pkg/tunnel/config"
	ccvalidation "github.com/kubeedge/kubeedge/pkg/apis/componentconfig/cloudcore/v1alpha1/validation"
)

func ValidateEdgeMeshServerConfiguration(c *config.EdgeMeshServerConfig) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, ccvalidation.ValidateKubeAPIConfig(*c.KubeAPIConfig)...)
	allErrs = append(allErrs, ValidateModuleTunnelServer(*c.Modules.TunnelServer)...)
	return allErrs
}

func ValidateModuleTunnelServer(c tunnelconfig.TunnelServerConfig) field.ErrorList {
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
