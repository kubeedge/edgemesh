package validation

import (
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/kubeedge/edgemesh/agent/cmd/edgemesh-agent/app/config"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/cloudcore/v1alpha1/validation"
)

func ValidateEdgeMeshAgentConfiguration(c *config.EdgeMeshAgentConfig) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, validation.ValidateKubeAPIConfig(*c.KubeAPIConfig)...)
	return allErrs
}
