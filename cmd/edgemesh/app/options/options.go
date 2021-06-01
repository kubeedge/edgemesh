package options

import (
	"path"

	config "github.com/kubeedge/edgemesh/pkg/apis/componentconfig/edgemesh/v1alpha1"
	"github.com/kubeedge/edgemesh/pkg/common/constants"
	cliflag "k8s.io/component-base/cli/flag"
)

type EdgeMeshOptions struct {
	ConfigFile string
}

func NewEdgeMeshOptions() *EdgeMeshOptions {
	return &EdgeMeshOptions{
		ConfigFile: path.Join(constants.DefaultConfigDir, "edgemesh.yaml"),
	}
}

func (e *EdgeMeshOptions) Flags() (fss cliflag.NamedFlagSets) {
	fs := fss.FlagSet("global")
	fs.StringVar(&e.ConfigFile, "config", e.ConfigFile, "The path to the configuration file. Flags override values in this file.")
	return
}

func (e *EdgeMeshOptions) Config() (*config.EdgeMeshConfig, error) {
	cfg := config.NewDefaultEdgeMeshConfig()
	if err := cfg.Parse(e.ConfigFile); err != nil {
		return nil, err
	}
	return cfg, nil
}
