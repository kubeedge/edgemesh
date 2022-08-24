package tunnel

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kubeedge/beehive/pkg/core"
	"github.com/libp2p/go-libp2p"
	p2phost "github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/routing"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	noise "github.com/libp2p/go-libp2p-noise"
	libp2ptlsca "github.com/libp2p/go-libp2p-tls"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

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

var Agent *EdgeTunnel

// EdgeTunnel is used for solving cross subset communication
type EdgeTunnel struct {
	Config     *config.EdgeTunnelConfig
	ProxySvc   *proxy.ProxyService
	Mode       TunnelMode
	kubeClient kubernetes.Interface

	peerHost     p2phost.Host
	peerMapMutex sync.Mutex // protect peerMap
	peerMap      map[string]*peer.AddrInfo

	nodeSynced   cache.InformerSynced
	resyncPeriod time.Duration

	stopCh chan struct{}
}

func newEdgeTunnel(c *config.EdgeTunnelConfig, ifm *informers.Manager, mode TunnelMode) (*EdgeTunnel, error) {
	Agent = &EdgeTunnel{Config: c}
	if !c.Enable {
		return Agent, nil
	}
	Agent.Config.NodeName = util.FetchNodeName()

	controller.Init(ifm)

	aclManager := security.NewManager(c.Security)

	aclManager.Start()

	/**
	TODO: 修改为Nodename 加密
	这个改为：
	priv, err := GenerateKeyPairWithString(HostnameOrDie())
	if err != nil {
		panic(err)
	}
	*/
	privateKey, err := aclManager.GetPrivateKey()
	if err != nil {
		return nil, fmt.Errorf("failed to get private key: %w", err)
	}

	/**
	TODO: 添加参数
	主要是l
	*/

	ctx := context.Background()
	var idht *dht.IpfsDHT
	opts := []libp2p.Option{
		libp2p.ListenAddrStrings(util.GenerateMultiAddr(c.Transport, "0.0.0.0", c.ListenPort)),
		util.GenerateTransportOption(c.Transport),
		libp2p.ForceReachabilityPrivate(),
		libp2p.Identity(privateKey),
		/**
		以下参数并不必须
		按照老师x
		*/

		libp2p.Security(libp2ptlsca.ID, libp2ptlsca.New),
		libp2p.Security(noise.ID, noise.New),
		libp2p.DefaultTransports,
		libp2p.NATPortMap(),
		libp2p.Routing(func(h p2phost.Host) (routing.PeerRouting, error) {
			idht, err = dht.New(ctx, h)
			return idht, err
		}),
		libp2p.EnableAutoRelay(),
		libp2p.EnableNATService(),
		libp2p.EnableHolePunching(),
		libp2p.ForceReachabilityPublic(),
	}

	//if c.Security.Enable {
	//	if err := libp2ptlsca.EnableCAEncryption(c.Security.TLSCAFile, c.Security.TLSCertFile,
	//		c.Security.TLSPrivateKeyFile); err != nil {
	//		return nil, fmt.Errorf("go-libp2p-tls: enable ca encryption err: %w", err)
	//	}
	//	opts = append(opts, libp2p.Security(libp2ptlsca.ID, libp2ptlsca.New))
	//} else {
	//	opts = append(opts, libp2p.NoSecurity)
	//}

	if c.EnableHolePunch {
		opts = append(opts, libp2p.EnableHolePunching())
	}

	h, err := libp2p.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to start tunnel agent: %w", err)
	}
	for _, addr := range h.Addrs() {
		klog.Infof("Listening on %s/p2p/%s", addr, h.ID().Pretty())
	}

	Agent.resyncPeriod = 15 * time.Minute
	Agent.kubeClient = ifm.GetKubeClient()
	Agent.peerMap = make(map[string]*peer.AddrInfo)
	Agent.stopCh = make(chan struct{})
	Agent.peerHost = h
	Agent.ProxySvc = proxy.NewProxyService(h)
	Agent.Mode = mode
	klog.V(4).Infof("tunnel agent mode is %v", mode)

	if mode == ServerClientMode {
		h.SetStreamHandler(proxy.ProxyProtocol, Agent.ProxySvc.ProxyStreamHandler)
	}

	return Agent, nil
}

// Register register edgetunnel to beehive modules
func Register(c *config.EdgeTunnelConfig, ifm *informers.Manager, mode TunnelMode) error {
	agent, err := newEdgeTunnel(c, ifm, mode)
	if err != nil {
		return fmt.Errorf("register module edgeTunnel error: %v", err)
	}
	core.Register(agent)
	return nil
}

// Name of edgetunnel
func (t *EdgeTunnel) Name() string {
	return modules.EdgeTunnelModuleName
}

// Group of edgetunnel
func (t *EdgeTunnel) Group() string {
	return modules.EdgeTunnelModuleName
}

// Enable indicates whether enable this module
func (t *EdgeTunnel) Enable() bool {
	return t.Config.Enable
}

// Start edgetunnel
func (t *EdgeTunnel) Start() {
	t.Run()
}
