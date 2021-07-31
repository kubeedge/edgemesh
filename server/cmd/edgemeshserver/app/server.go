package app

import (
	"fmt"

	"github.com/kubeedge/beehive/pkg/core"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/edgemesh/pkg/apis/componentconfig/edgemesh-server/v1alpha1"
	"github.com/kubeedge/edgemesh/pkg/common/client"
	"github.com/kubeedge/edgemesh/pkg/common/informers"
	"github.com/kubeedge/edgemesh/server/cmd/edgemeshserver/app/options"
	"github.com/kubeedge/edgemesh/server/pkg/tunnel"
	"github.com/kubeedge/kubeedge/pkg/version/verflag"
	"github.com/spf13/cobra"
	"k8s.io/client-go/pkg/version"
	cliflag "k8s.io/component-base/cli/flag"
	"k8s.io/component-base/cli/globalflag"
	"k8s.io/component-base/term"
	"k8s.io/klog/v2"
)

func NewEdgeMeshServerCommand() *cobra.Command {
	opts := options.NewEdgeMeshServerOptions()
	cmd := &cobra.Command{
		Use:  "edgemesh-server",
		Long: "Edgemesh-Server is the part .....",
		Run: func(cmd *cobra.Command, args []string) {
			verflag.PrintAndExitIfRequested()
			cliflag.PrintFlags(cmd.Flags())

			edgeMeshServerConfig, err := opts.Config()
			if err != nil {
				klog.Fatal(err)
			}

			// To help debugging , immediately log version
			klog.Infof("Version: %+v", version.Get())
			client.InitEdgeMeshClient(edgeMeshServerConfig.KubeAPIConfig)
			gis := informers.GetInformersManager()
			registerModules(edgeMeshServerConfig)

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

func registerModules(c *v1alpha1.Config) {
	tunnel.Register(c.Modules.Tunnel)
}
