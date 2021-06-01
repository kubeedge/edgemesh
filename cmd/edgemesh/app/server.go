package app

import (
	"fmt"

	"github.com/kubeedge/kubeedge/pkg/version/verflag"
	"github.com/spf13/cobra"
	cliflag "k8s.io/component-base/cli/flag"
	"k8s.io/component-base/cli/globalflag"
	"k8s.io/component-base/term"
	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/edgemesh/cmd/edgemesh/app/options"
	"github.com/kubeedge/edgemesh/pkg/apis/componentconfig/edgemesh/v1alpha1"
	"github.com/kubeedge/edgemesh/pkg/common/client"
	"github.com/kubeedge/edgemesh/pkg/common/informers"
	"github.com/kubeedge/edgemesh/pkg/controller"
	"github.com/kubeedge/edgemesh/pkg/networking"
)

func NewEdgeMeshCommand() *cobra.Command {
	opts := options.NewEdgeMeshOptions()
	cmd := &cobra.Command{
		Use: "edgemesh",
		Long: `EdgeMesh is a part of KubeEdge, and provides a simple network solution
for the inter-communications between services at edge scenarios.`,
		Run: func(cmd *cobra.Command, args []string) {
			verflag.PrintAndExitIfRequested()
			cliflag.PrintFlags(cmd.Flags())

			config, err := opts.Config()
			if err != nil {
				klog.Fatal(err)
			}

			client.InitEdgeMeshClient(config.KubeAPIConfig)
			gis := informers.GetInformersManager()
			registerModules(config)

			// start all modules
			core.StartModules()
			gis.Start(beehiveContext.Done())
			core.GracefulShutdown()
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

// registerModules register all the modules started in edgemesh
func registerModules(c *v1alpha1.EdgeMeshConfig) {
	controller.Register(c.Modules.Controller)
	networking.Register(c.Modules.Networking)
}
