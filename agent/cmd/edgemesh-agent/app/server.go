package app

import (
	"fmt"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/wait"
	cliflag "k8s.io/component-base/cli/flag"
	"k8s.io/component-base/cli/globalflag"
	"k8s.io/component-base/term"
	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/edgemesh/agent/cmd/edgemesh-agent/app/config"
	"github.com/kubeedge/edgemesh/agent/cmd/edgemesh-agent/app/config/validation"
	"github.com/kubeedge/edgemesh/agent/cmd/edgemesh-agent/app/options"
	"github.com/kubeedge/edgemesh/agent/pkg/chassis"
	"github.com/kubeedge/edgemesh/agent/pkg/dns"
	"github.com/kubeedge/edgemesh/agent/pkg/gateway"
	"github.com/kubeedge/edgemesh/agent/pkg/proxy"
	"github.com/kubeedge/edgemesh/agent/pkg/tunnel"
	"github.com/kubeedge/edgemesh/common/informers"
	commonutil "github.com/kubeedge/edgemesh/common/util"
	"github.com/kubeedge/kubeedge/pkg/util"
	"github.com/kubeedge/kubeedge/pkg/util/flag"
	"github.com/kubeedge/kubeedge/pkg/version"
	"github.com/kubeedge/kubeedge/pkg/version/verflag"
)

func NewEdgeMeshAgentCommand() *cobra.Command {
	opts := options.NewEdgeMeshAgentOptions()
	cmd := &cobra.Command{
		Use: "edgemesh-agent",
		Long: `edgemesh-agent is a part of EdgeMesh which provides a simple network solution
for the inter-communications between services at edge scenarios.`,
		Run: func(cmd *cobra.Command, args []string) {
			verflag.PrintAndExitIfRequested()
			flag.PrintMinConfigAndExitIfRequested(config.NewEdgeMeshAgentConfig())
			flag.PrintDefaultConfigAndExitIfRequested(config.NewEdgeMeshAgentConfig())
			flag.PrintFlags(cmd.Flags())

			if errs := opts.Validate(); len(errs) > 0 {
				klog.Exit(util.SpliceErrors(errs))
			}

			agentCfg, err := opts.Config()
			if err != nil {
				klog.Exit(err)
			}

			if errs := validation.ValidateEdgeMeshAgentConfiguration(agentCfg); len(errs) > 0 {
				klog.Exit(util.SpliceErrors(errs.ToAggregate().Errors()))
			}

			klog.Infof("Version: %+v", version.Get())
			if err = Run(agentCfg); err != nil {
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

// Run runs EdgeMesh Agent
func Run(cfg *config.EdgeMeshAgentConfig) error {
	trace := 1

	klog.Infof("[%d] New informers manager", trace)
	ifm, err := informers.NewManager(cfg.KubeAPIConfig)
	if err != nil {
		return err
	}
	trace++

	klog.Infof("[%d] Prepare agent to run", trace)
	if err = prepareRun(cfg); err != nil {
		return err
	}
	klog.Infof("edgemesh-agent running on %s", cfg.CommonConfig.Mode)
	trace++

	// NOTE: we only install go-chassis when the gateway module enabled.
	if cfg.Modules.EdgeGatewayConfig.Enable {
		klog.Infof("[%d] Install go-chassis plugins", trace)
		chassis.Install(cfg.GoChassisConfig, ifm)
		trace++
	}

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

	klog.Infof("edgemesh-agent exited")
	return nil
}

// registerModules register all the modules started in edgemesh-agent
func registerModules(c *config.EdgeMeshAgentConfig, ifm *informers.Manager) []error {
	var errs []error
	if err := dns.Register(c.Modules.EdgeDNSConfig, ifm); err != nil {
		errs = append(errs, err)
	}
	if err := proxy.Register(c.Modules.EdgeProxyConfig, ifm); err != nil {
		errs = append(errs, err)
	}
	if err := gateway.Register(c.Modules.EdgeGatewayConfig, ifm); err != nil {
		errs = append(errs, err)
	}

	mode := tunnel.ServerClientMode
	if c.Modules.EdgeGatewayConfig.Enable {
		mode = tunnel.ClientMode
	}

	if err := tunnel.Register(c.Modules.EdgeTunnelConfig, mode); err != nil {
		errs = append(errs, err)
	}
	return errs
}

// prepareRun prepares edgemesh-agent to run
func prepareRun(c *config.EdgeMeshAgentConfig) error {
	// if the user sets KubeConfig or Master and Master is not equal to EdgeApiServer, enter the debug mode
	if c.KubeAPIConfig.KubeConfig != "" || c.KubeAPIConfig.Master != "" &&
		c.KubeAPIConfig.Master != config.DefaultEdgeApiServer {
		c.CommonConfig.Mode = config.DebugMode
	}

	// set dns and proxy modules listenInterface
	if c.Modules.EdgeDNSConfig.Enable || c.Modules.EdgeProxyConfig.Enable {
		if err := commonutil.CreateEdgeMeshDevice(c.CommonConfig.DummyDeviceName, c.CommonConfig.DummyDeviceIP); err != nil {
			return fmt.Errorf("failed to create edgemesh device %s: %w", c.CommonConfig.DummyDeviceName, err)
		}
		c.Modules.EdgeDNSConfig.ListenInterface = c.CommonConfig.DummyDeviceName
		c.Modules.EdgeProxyConfig.ListenInterface = c.CommonConfig.DummyDeviceName
	}

	// set dns module mode
	if c.Modules.EdgeDNSConfig.Enable {
		c.Modules.EdgeDNSConfig.Mode = c.CommonConfig.Mode
		c.Modules.EdgeDNSConfig.KubeAPIConfig = c.KubeAPIConfig
	}

	return nil
}
