package validation

import (
	"k8s.io/apimachinery/pkg/util/validation/field"

	ccvalidation "github.com/kubeedge/kubeedge/pkg/apis/componentconfig/cloudcore/v1alpha1/validation"

	"github.com/kubeedge/edgemesh/agent/cmd/edgemesh-agent/app/config"
)

func ValidateEdgeMeshAgentConfiguration(c *config.EdgeMeshAgentConfig) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, ccvalidation.ValidateKubeAPIConfig(*c.KubeAPIConfig)...)
	return allErrs
}
