package tunnel

import (
	"context"
	"fmt"

	"github.com/libp2p/go-libp2p"
	circuit "github.com/libp2p/go-libp2p-circuit"
	"github.com/libp2p/go-libp2p-core/host"
	libp2ptlsca "github.com/libp2p/go-libp2p-tls"
	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/edgemesh/agent/pkg/tunnel/config"
	"github.com/kubeedge/edgemesh/agent/pkg/tunnel/controller"
	"github.com/kubeedge/edgemesh/agent/pkg/tunnel/proxy"
	"github.com/kubeedge/edgemesh/common/informers"
	"github.com/kubeedge/edgemesh/common/modules"
	"github.com/kubeedge/edgemesh/common/security"
	"github.com/kubeedge/edgemesh/common/util"
)

type TunnelMode string

const (
	ClientMode       TunnelMode = "ClientOnly"
	ServerClientMode TunnelMode = "ServerAndClient"
	UnknownMode      TunnelMode = "Unknown"
)

var Agent *TunnelAgent

// TunnelAgent is used for solving cross subset communication
type TunnelAgent struct {
	Config   *config.TunnelAgentConfig
	Host     host.Host
	ProxySvc *proxy.ProxyService
	Mode     TunnelMode
}

func newTunnelAgent(c *config.TunnelAgentConfig, ifm *informers.Manager, mode TunnelMode) (*TunnelAgent, error) {
	Agent = &TunnelAgent{Config: c}
	if !c.Enable {
		return Agent, nil
	}
	Agent.Config.NodeName = util.FetchNodeName()

	controller.Init(ifm)

	aclManager := security.NewManager(c.Security)

	aclManager.Start()

	privateKey, err := aclManager.GetPrivateKey()
	if err != nil {
		return nil, fmt.Errorf("failed to get private key: %w", err)
	}

	opts := []libp2p.Option{
		libp2p.ListenAddrStrings(util.GenerateMultiAddr(c.Transport, "0.0.0.0", c.ListenPort)),
		util.GenerateTransportOption(c.Transport),
		libp2p.EnableRelay(circuit.OptActive),
		libp2p.ForceReachabilityPrivate(),
		libp2p.Identity(privateKey),
	}

	if c.Security.Enable {
		if err := libp2ptlsca.EnableCAEncryption(c.Security.TLSCAFile, c.Security.TLSCertFile,
			c.Security.TLSPrivateKeyFile); err != nil {
			return nil, fmt.Errorf("go-libp2p-tls: enable ca encryption err: %w", err)
		}
		opts = append(opts, libp2p.Security(libp2ptlsca.ID, libp2ptlsca.New))
	} else {
		opts = append(opts, libp2p.NoSecurity)
	}

	if c.EnableHolePunch {
		opts = append(opts, libp2p.EnableHolePunching())
	}

	h, err := libp2p.New(context.Background(), opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to start tunnel agent: %w", err)
	}
	for _, addr := range h.Addrs() {
		fmt.Println("Listening on", addr)
	}

	Agent.Host = h
	Agent.ProxySvc = proxy.NewProxyService(h)
	Agent.Mode = mode
	klog.V(4).Infof("tunnel agent mode is %v", mode)

	if mode == ServerClientMode {
		h.SetStreamHandler(proxy.ProxyProtocol, Agent.ProxySvc.ProxyStreamHandler)
	}

	return Agent, nil
}

// Register register tunnelagent to beehive modules
func Register(c *config.TunnelAgentConfig, ifm *informers.Manager, mode TunnelMode) error {
	agent, err := newTunnelAgent(c, ifm, mode)
	if err != nil {
		return fmt.Errorf("register module tunnelagent error: %v", err)
	}
	core.Register(agent)
	return nil
}

// Name of tunnelagent
func (t *TunnelAgent) Name() string {
	return modules.TunnelAgentModuleName
}

// Group of tunnelagent
func (t *TunnelAgent) Group() string {
	return modules.TunnelAgentModuleName
}

// Enable indicates whether enable this module
func (t *TunnelAgent) Enable() bool {
	return t.Config.Enable
}

// Start tunnelserver
func (t *TunnelAgent) Start() {
	t.Run()
}
