package app

import (
	"fmt"
	"os"

	"github.com/kubeedge/beehive/pkg/core"
	kubeedgeutil "github.com/kubeedge/kubeedge/pkg/util"
	"github.com/kubeedge/kubeedge/pkg/util/flag"
	"github.com/kubeedge/kubeedge/pkg/version"
	"github.com/kubeedge/kubeedge/pkg/version/verflag"
	"github.com/spf13/cobra"
	cliflag "k8s.io/component-base/cli/flag"
	"k8s.io/component-base/cli/globalflag"
	"k8s.io/component-base/term"
	"k8s.io/klog/v2"

	"github.com/kubeedge/edgemesh/cmd/edgemesh-agent/app/options"
	"github.com/kubeedge/edgemesh/pkg/apis/config/defaults"
	"github.com/kubeedge/edgemesh/pkg/apis/config/v1alpha1"
	"github.com/kubeedge/edgemesh/pkg/apis/config/v1alpha1/validation"
	"github.com/kubeedge/edgemesh/pkg/clients"
	"github.com/kubeedge/edgemesh/pkg/dns"
	"github.com/kubeedge/edgemesh/pkg/proxy"
	"github.com/kubeedge/edgemesh/pkg/tunnel"
	netutil "github.com/kubeedge/edgemesh/pkg/util/net"
)

func NewEdgeMeshAgentCommand() *cobra.Command {
	opts := options.NewEdgeMeshAgentOptions()
	cmd := &cobra.Command{
		Use:  "edgemesh-agent",
		Long: `edgemesh-agent is the data plane component of EdgeMesh.`,
		Run: func(cmd *cobra.Command, args []string) {
			verflag.PrintAndExitIfRequested()
			flag.PrintMinConfigAndExitIfRequested(v1alpha1.NewDefaultEdgeMeshAgentConfig())
			flag.PrintDefaultConfigAndExitIfRequested(v1alpha1.NewDefaultEdgeMeshAgentConfig())
			flag.PrintFlags(cmd.Flags())

			if errs := opts.Validate(); len(errs) > 0 {
				klog.Exit(kubeedgeutil.SpliceErrors(errs))
			}

			cfg, err := opts.Config()
			if err != nil {
				klog.Exit(err)
			}

			if errs := validation.ValidateEdgeMeshAgentConfiguration(cfg); len(errs) > 0 {
				klog.Exit(kubeedgeutil.SpliceErrors(errs.ToAggregate().Errors()))
			}

			klog.Infof("Version: %+v", version.Get())
			if err = Run(cfg); err != nil {
				klog.Exit("run edgemesh-agent failed: ", err)
			}
		},
	}
	fs := cmd.Flags()
	namedFs := opts.Flags()
	verflag.AddFlags(namedFs.FlagSet("global"))
	flag.AddFlags(namedFs.FlagSet("global"))
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

// Run runs edgemesh-agent
func Run(cfg *v1alpha1.EdgeMeshAgentConfig) error {
	trace := 1

	klog.Infof("[%d] Prepare agent to run", trace)
	if err := prepareRun(cfg); err != nil {
		return err
	}
	klog.Infof("edgemesh-agent running on %s", cfg.KubeAPIConfig.Mode)
	trace++

	klog.Infof("[%d] New clients", trace)
	cli, err := clients.NewClients(cfg.KubeAPIConfig)
	if err != nil {
		return err
	}
	trace++

	klog.Infof("[%d] Register beehive modules", trace)
	if errs := registerModules(cfg, cli); len(errs) > 0 {
		return fmt.Errorf(kubeedgeutil.SpliceErrors(errs))
	}
	trace++

	klog.Infof("[%d] Start all modules", trace)
	core.Run()

	klog.Infof("edgemesh-agent exited")
	return nil
}

// registerModules register all the modules started in edgemesh-agent
func registerModules(c *v1alpha1.EdgeMeshAgentConfig, cli *clients.Clients) []error {
	var errs []error
	if err := dns.Register(c.Modules.EdgeDNSConfig, cli); err != nil {
		errs = append(errs, err)
	}
	if err := proxy.Register(c.Modules.EdgeProxyConfig, cli); err != nil {
		errs = append(errs, err)
	}
	if err := tunnel.Register(c.Modules.EdgeTunnelConfig); err != nil {
		errs = append(errs, err)
	}
	return errs
}

// prepareRun prepares edgemesh-agent to run
func prepareRun(c *v1alpha1.EdgeMeshAgentConfig) error {
	// If in the edge mode and the user does not configure KubeAPIConfig.Master,
	// set KubeAPIConfig.Master to the value of CommonConfig.MetaServerAddress
	if c.KubeAPIConfig.Mode == defaults.EdgeMode && c.KubeAPIConfig.Master == defaults.MetaServerAddress {
		c.KubeAPIConfig.Master = c.KubeAPIConfig.MetaServerAddress
	}

	// If the user sets KubeConfig or Master and Master is not equal to
	// EdgeCore's metaServer address, then enter the debug mode
	if c.KubeAPIConfig.KubeConfig != "" || c.KubeAPIConfig.Master != "" &&
		c.KubeAPIConfig.Master != c.KubeAPIConfig.MetaServerAddress {
		c.KubeAPIConfig.Mode = defaults.DebugMode
	}

	// Set dns and proxy modules listenInterface
	err := netutil.CreateEdgeMeshDevice(c.CommonConfig.BridgeDeviceName, c.CommonConfig.BridgeDeviceIP)
	if err != nil {
		return fmt.Errorf("failed to create edgemesh device %s: %w", c.CommonConfig.BridgeDeviceName, err)
	}
	c.Modules.EdgeDNSConfig.ListenInterface = c.CommonConfig.BridgeDeviceName
	c.Modules.EdgeProxyConfig.ListenInterface = c.CommonConfig.BridgeDeviceName

	// Set dns module KubeAPIConfig
	c.Modules.EdgeDNSConfig.KubeAPIConfig = c.KubeAPIConfig

	// Set node name and namespace
	nodeName, exists := os.LookupEnv("NODE_NAME")
	if !exists {
		return fmt.Errorf("env NODE_NAME not exist")
	}
	namespace, exists := os.LookupEnv("NAMESPACE")
	if !exists {
		return fmt.Errorf("env NAMESPACE not exist")
	}
	c.Modules.EdgeProxyConfig.LoadBalancer.NodeName = nodeName
	c.Modules.EdgeProxyConfig.Socks5Proxy.NodeName = nodeName
	c.Modules.EdgeProxyConfig.Socks5Proxy.Namespace = namespace
	c.Modules.EdgeTunnelConfig.NodeName = nodeName

	// Set tunnel module mode
	c.Modules.EdgeTunnelConfig.Mode = defaults.ServerClientMode

	return nil
}
