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

type EdgeMeshAgentOptions struct {
	ConfigFile string
}

func NewEdgeMeshAgentOptions() *EdgeMeshAgentOptions {
	return &EdgeMeshAgentOptions{
		ConfigFile: path.Join(defaults.ConfigDir, defaults.EdgeMeshAgentConfigName),
	}
}

func (o *EdgeMeshAgentOptions) Flags() (fss cliflag.NamedFlagSets) {
	fs := fss.FlagSet("global")
	fs.StringVar(&o.ConfigFile, "config-file", o.ConfigFile, "The path to the configuration file. Flags override values in this file.")
	return
}

func (o *EdgeMeshAgentOptions) Validate() []error {
	var errs []error
	if !validation.FileIsExist(o.ConfigFile) {
		errs = append(errs, field.Required(field.NewPath("config-file"),
			fmt.Sprintf("config file %v not exist", o.ConfigFile)))
	}
	return errs
}

func (o *EdgeMeshAgentOptions) Parse(cfg *v1alpha1.EdgeMeshAgentConfig) error {
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

// Config generates *v1alpha1.EdgeMeshAgentConfig
func (o *EdgeMeshAgentOptions) Config() (*v1alpha1.EdgeMeshAgentConfig, error) {
	cfg := v1alpha1.NewDefaultEdgeMeshAgentConfig(o.ConfigFile)
	if err := o.Parse(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
