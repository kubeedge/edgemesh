package options

import (
	"path"

	config "github.com/kubeedge/edgemesh/pkg/apis/componentconfig/edgemesh/v1alpha1"
	"github.com/kubeedge/edgemesh/pkg/common/constants"
)

type EdgeMeshOptions struct {
	ConfigFile string
}

func NewEdgeMeshOptions() *EdgeMeshOptions {
	return &EdgeMeshOptions{
		ConfigFile: path.Join(constants.DefaultConfigDir, "edgemesh.yaml"),
	}
}

func (e *EdgeMeshOptions) Config() (*config.EdgeMeshConfig, error) {
	cfg := config.NewDefaultEdgeMeshConfig()
	if err := cfg.Parse(e.ConfigFile); err != nil {
		return nil, err
	}
	return cfg, nil
}
