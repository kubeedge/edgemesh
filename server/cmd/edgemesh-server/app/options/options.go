package options

import (
	"fmt"
	"path"

	"k8s.io/apimachinery/pkg/util/validation/field"
	cliflag "k8s.io/component-base/cli/flag"

	"github.com/kubeedge/edgemesh/server/cmd/edgemesh-server/app/config"
	"github.com/kubeedge/kubeedge/common/constants"
	"github.com/kubeedge/kubeedge/pkg/util/validation"
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

func (e *EdgeMeshServerOptions) Validate() []error {
	var errs []error
	if !validation.FileIsExist(e.ConfigFile) {
		errs = append(errs, field.Required(field.NewPath("config-file"),
			fmt.Sprintf("config file %v not exist", e.ConfigFile)))
	}
	return errs
}

func (e *EdgeMeshServerOptions) Config() (*config.EdgeMeshServerConfig, error) {
	cfg := config.NewEdgeMeshServerConfig()
	if err := cfg.Parse(e.ConfigFile); err != nil {
		return nil, err
	}
	return cfg, nil
}
