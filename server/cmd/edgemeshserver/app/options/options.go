package options

import (
	"path"

	"github.com/kubeedge/edgemesh/pkg/apis/componentconfig/edgemesh-server/v1alpha1"
	cliflag "k8s.io/component-base/cli/flag"

	"github.com/kubeedge/kubeedge/common/constants"
)

type EdgeMeshServerOptions struct {
	ConfigFile string
}

func NewEdgeMeshServerOptions() *EdgeMeshServerOptions {
	return &EdgeMeshServerOptions{
		ConfigFile: path.Join(constants.DefaultConfigDir, "edgemesh-server.yaml"),
	}
}

func (e *EdgeMeshServerOptions) Flags() (fss cliflag.NamedFlagSets) {
	fs := fss.FlagSet("global")
	fs.StringVar(&e.ConfigFile, "config", e.ConfigFile, "The path to the configuration file. Flags override values in this file.")
	return
}


func (e *EdgeMeshServerOptions) Config() (*v1alpha1.Config, error) {
	cfg := v1alpha1.NewDefaultEdgeMeshServerConfig()
	if err := cfg.Parse(e.ConfigFile); err != nil {
		return nil, err
	}
	return cfg, nil
}
