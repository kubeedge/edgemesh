package app

import (
	"fmt"
	"k8s.io/apimachinery/pkg/util/wait"

	"k8s.io/client-go/pkg/version"
	cliflag "k8s.io/component-base/cli/flag"
	"k8s.io/component-base/cli/globalflag"
	"k8s.io/component-base/term"
	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/edgemesh/common/informers"
	"github.com/kubeedge/edgemesh/server/cmd/edgemesh-server/app/config"
	"github.com/kubeedge/edgemesh/server/cmd/edgemesh-server/app/config/validation"
	"github.com/kubeedge/edgemesh/server/cmd/edgemesh-server/app/options"
	"github.com/kubeedge/edgemesh/server/pkg/tunnel"
	"github.com/kubeedge/kubeedge/pkg/util"
	"github.com/kubeedge/kubeedge/pkg/util/flag"
	"github.com/kubeedge/kubeedge/pkg/version/verflag"
	"github.com/spf13/cobra"
)

func NewEdgeMeshServerCommand() *cobra.Command {
	opts := options.NewEdgeMeshServerOptions()
	cmd := &cobra.Command{
		Use:  "edgemesh-server",
		Long: `edgemesh-server is a part of EdgeMesh, and provides signal and relay service for edgemesh-agents.`,
		Run: func(cmd *cobra.Command, args []string) {
			verflag.PrintAndExitIfRequested()
			flag.PrintMinConfigAndExitIfRequested(config.NewEdgeMeshServerConfig())
			flag.PrintDefaultConfigAndExitIfRequested(config.NewEdgeMeshServerConfig())
			flag.PrintFlags(cmd.Flags())

			if errs := opts.Validate(); len(errs) > 0 {
				klog.Fatal(util.SpliceErrors(errs))
			}

			edgeMeshServerConfig, err := opts.Config()
			if err != nil {
				klog.Fatal(err)
			}

			if errs := validation.ValidateEdgeMeshServerConfiguration(edgeMeshServerConfig); len(errs) > 0 {
				klog.Fatal(util.SpliceErrors(errs.ToAggregate().Errors()))
			}

			klog.Infof("Version: %+v", version.Get())
			if err = Run(edgeMeshServerConfig); err != nil {
				klog.Fatalf("run edgemesh-server failed: %v", err)
			}
		},
	}
	fs := cmd.Flags()
	namedFs := opts.Flags()
	verflag.AddFlags(namedFs.FlagSet("global"))
	globalflag.AddGlobalFlags(namedFs.FlagSet("global"), cmd.Name())
	for _, f := range namedFs.FlagSets {
		fs.AddFlagSet(f)
	}

	usageFmt := "Usage:\n  %s\n"
	cols, _, _ := term.TerminalSize(cmd.OutOrStdout())
	cmd.SetUsageFunc(func(cmd *cobra.Command) error {
		fmt.Fprintf(cmd.OutOrStderr(), usageFmt, cmd.UseLine())
		cliflag.PrintSections(cmd.OutOrStderr(), namedFs, cols)
		return nil
	})
	cmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		fmt.Fprintf(cmd.OutOrStdout(), "%s\n\n"+usageFmt, cmd.Long, cmd.UseLine())
		cliflag.PrintSections(cmd.OutOrStdout(), namedFs, cols)
	})
	return cmd
}

//Run runs EdgeMesh Server
func Run(cfg *config.EdgeMeshServerConfig) error {
	trace := 1

	klog.Infof("[%d] New informers manager", trace)
	ifm, err := informers.NewManager(cfg.KubeAPIConfig)
	if err != nil {
		return err
	}
	trace++

	klog.Infof("[%d] Register beehive modules", trace)
	if errs := registerModules(cfg, ifm); len(errs) > 0 {
		return fmt.Errorf(util.SpliceErrors(errs))
	}
	trace++

	klog.Infof("[%d] Start informers manager", trace)
	ifm.Start(wait.NeverStop)
	trace++

	klog.Infof("[%d] Start all modules", trace)
	core.Run()

	klog.Infof("edgemesh-server exited")
	return nil
}

// registerModules register all the modules started in edgemesh-server
func registerModules(c *config.EdgeMeshServerConfig, ifm *informers.Manager) []error {
	var errs []error
	if err := tunnel.Register(c.Modules.TunnelServer, ifm); err != nil {
		errs = append(errs, err)
	}
	return nil
}
