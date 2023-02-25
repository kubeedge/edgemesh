package options

import (
	"fmt"
	"io/ioutil"
	"path"

	"k8s.io/apimachinery/pkg/util/validation/field"
	cliflag "k8s.io/component-base/cli/flag"
	"k8s.io/klog/v2"
	"sigs.k8s.io/yaml"

	"github.com/kubeedge/edgemesh/pkg/apis/config/defaults"
	"github.com/kubeedge/edgemesh/pkg/apis/config/v1alpha1"
	"github.com/kubeedge/kubeedge/pkg/util/validation"
)

type EdgeMeshGatewayOptions struct {
	ConfigFile string
}

func NewEdgeMeshGatewayOptions() *EdgeMeshGatewayOptions {
	return &EdgeMeshGatewayOptions{
		ConfigFile: path.Join(defaults.ConfigDir, defaults.EdgeMeshGatewayConfigName),
	}
}

func (o *EdgeMeshGatewayOptions) Flags() (fss cliflag.NamedFlagSets) {
	fs := fss.FlagSet("global")
	fs.StringVar(&o.ConfigFile, "config-file", o.ConfigFile, "The path to the configuration file. Flags override values in this file.")
	return
}

func (o *EdgeMeshGatewayOptions) Validate() []error {
	var errs []error
	if !validation.FileIsExist(o.ConfigFile) {
		errs = append(errs, field.Required(field.NewPath("config-file"),
			fmt.Sprintf("config file %v not exist", o.ConfigFile)))
	}
	return errs
}

func (o *EdgeMeshGatewayOptions) Parse(cfg *v1alpha1.EdgeMeshGatewayConfig) error {
	data, err := ioutil.ReadFile(o.ConfigFile)
	if err != nil {
		klog.Errorf("Failed to read config file %s: %v", o.ConfigFile, err)
		return err
	}
	err = yaml.Unmarshal(data, cfg)
	if err != nil {
		klog.Errorf("Failed to unmarshal config file %s: %v", o.ConfigFile, err)
		return err
	}
	return nil
}

// Config generates *v1alpha1.EdgeMeshGatewayConfig
func (o *EdgeMeshGatewayOptions) Config() (*v1alpha1.EdgeMeshGatewayConfig, error) {
	cfg := v1alpha1.NewDefaultEdgeMeshGatewayConfig(o.ConfigFile)
	if err := o.Parse(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
