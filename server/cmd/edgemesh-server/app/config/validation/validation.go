package validation

import (
	"github.com/kubeedge/edgemesh/server/cmd/edgemesh-server/app/config"
	ccvalidation "github.com/kubeedge/kubeedge/pkg/apis/componentconfig/cloudcore/v1alpha1/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func ValidateEdgeMeshServerConfiguration(c *config.EdgeMeshServerConfig) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, ccvalidation.ValidateKubeAPIConfig(*c.KubeAPIConfig)...)
	return allErrs
}
