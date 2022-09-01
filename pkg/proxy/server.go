package proxy

import (
	"errors"
	"fmt"
	"time"

	istioclientset "istio.io/client-go/pkg/clientset/versioned"
	istioinformers "istio.io/client-go/pkg/informers/externalversions"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	utilnet "k8s.io/apimachinery/pkg/util/net"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/proxy"
	"k8s.io/kubernetes/pkg/proxy/apis"
	proxyconfigapi "k8s.io/kubernetes/pkg/proxy/apis/config"
	utiliptables "k8s.io/kubernetes/pkg/util/iptables"
	"k8s.io/utils/exec"
	netutils "k8s.io/utils/net"

	"github.com/kubeedge/edgemesh/third_party/forked/kubernetes/pkg/proxy/config"
	"github.com/kubeedge/edgemesh/third_party/forked/kubernetes/pkg/proxy/userspace"
)

// Copy and update from https://github.com/kubernetes/kubernetes/blob/v1.23.0/cmd/kube-proxy/app/server.go and
// https://github.com/kubernetes/kubernetes/blob/v1.23.0/cmd/kube-proxy/app/server_others.go.

const (
	proxyModeUserspace = "userspace"

	// labelEdgeMeshServiceProxyName indicates that an alternative service
	// proxy will implement this Service.
	labelEdgeMeshServiceProxyName = "service.edgemesh.kubeedge.io/service-proxy-name"
)

// ProxyServer represents all the parameters required to start the Kubernetes proxy server.
type ProxyServer struct {
	Client            clientset.Interface
	IstioClient       istioclientset.Interface
	IptInterface      utiliptables.Interface
	execer            exec.Interface
	Proxier           proxy.Provider
	ProxyMode         string
	UseEndpointSlices bool
	ConfigSyncPeriod  time.Duration
}

// NewDefaultKubeProxyConfiguration new default kube-proxy config for edgemesh-agent runtime.
// TODO(Poorunga) Use container config for this.
func NewDefaultKubeProxyConfiguration(bindAddress string) *proxyconfigapi.KubeProxyConfiguration {
	return &proxyconfigapi.KubeProxyConfiguration{
		BindAddress: bindAddress,
		PortRange:   "",
		IPTables: proxyconfigapi.KubeProxyIPTablesConfiguration{
			SyncPeriod:    metav1.Duration{Duration: 30 * time.Second},
			MinSyncPeriod: metav1.Duration{Duration: time.Second},
		},
		UDPIdleTimeout:    metav1.Duration{Duration: 250 * time.Millisecond},
		NodePortAddresses: nil,
		ConfigSyncPeriod:  metav1.Duration{Duration: 15 * time.Minute},
	}
}

func newProxyServer(
	config *proxyconfigapi.KubeProxyConfiguration,
	cleanupAndExit bool,
	client clientset.Interface,
	istioClient istioclientset.Interface) (*ProxyServer, error) {
	if config == nil {
		return nil, errors.New("config is required")
	}

	// Create a iptables utils.
	execer := exec.New()
	iptInterface := utiliptables.New(execer, utiliptables.ProtocolIPv4)

	// We omit creation of pretty much everything if we run in cleanup mode
	if cleanupAndExit {
		return &ProxyServer{execer: execer}, nil
	}

	klog.V(0).Info("Using userspace Proxier.")

	// TODO this has side effects that should only happen when Run() is invoked.
	proxier, err := userspace.NewProxier(
		userspace.NewLoadBalancerEX(),
		netutils.ParseIPSloppy(config.BindAddress),
		iptInterface,
		execer,
		*utilnet.ParsePortRangeOrDie(config.PortRange),
		config.IPTables.SyncPeriod.Duration,
		config.IPTables.MinSyncPeriod.Duration,
		config.UDPIdleTimeout.Duration,
		config.NodePortAddresses,
	)
	if err != nil {
		return nil, fmt.Errorf("unable to create proxier: %v", err)
	}

	return &ProxyServer{
		Client:            client,
		IstioClient:       istioClient,
		IptInterface:      iptInterface,
		execer:            execer,
		Proxier:           proxier,
		ProxyMode:         proxyModeUserspace,
		ConfigSyncPeriod:  config.ConfigSyncPeriod.Duration,
		UseEndpointSlices: false,
	}, nil
}

// Run runs the specified ProxyServer.  This should never exit (unless CleanupAndExit is set).
func (s *ProxyServer) Run() error {
	noProxyName, err := labels.NewRequirement(apis.LabelServiceProxyName, selection.DoesNotExist, nil)
	if err != nil {
		return err
	}

	noEdgeMeshProxyName, err := labels.NewRequirement(labelEdgeMeshServiceProxyName, selection.DoesNotExist, nil)
	if err != nil {
		return err
	}

	noHeadlessEndpoints, err := labels.NewRequirement(v1.IsHeadlessService, selection.DoesNotExist, nil)
	if err != nil {
		return err
	}

	labelSelector := labels.NewSelector()
	labelSelector = labelSelector.Add(*noProxyName, *noEdgeMeshProxyName, *noHeadlessEndpoints)

	// Make informers that filter out objects that want a non-default service proxy.
	informerFactory := informers.NewSharedInformerFactoryWithOptions(s.Client, s.ConfigSyncPeriod,
		informers.WithTweakListOptions(func(options *metav1.ListOptions) {
			options.LabelSelector = labelSelector.String()
		}))
	istioInformerFactory := istioinformers.NewSharedInformerFactory(s.IstioClient, s.ConfigSyncPeriod)

	// Create configs (i.e. Watches for Services and Endpoints or EndpointSlices)
	// Note: RegisterHandler() calls need to happen before creation of Sources because sources
	// only notify on changes, and the initial update (on process start) may be lost if no handlers
	// are registered yet.
	serviceConfig := config.NewServiceConfig(informerFactory.Core().V1().Services(), s.ConfigSyncPeriod)
	serviceConfig.RegisterEventHandler(s.Proxier)
	go serviceConfig.Run(wait.NeverStop)

	if endpointsHandler, ok := s.Proxier.(config.EndpointsHandler); ok && !s.UseEndpointSlices {
		endpointsConfig := config.NewEndpointsConfig(informerFactory.Core().V1().Endpoints(), s.ConfigSyncPeriod)
		endpointsConfig.RegisterEventHandler(endpointsHandler)
		go endpointsConfig.Run(wait.NeverStop)
	} else {
		endpointSliceConfig := config.NewEndpointSliceConfig(informerFactory.Discovery().V1().EndpointSlices(), s.ConfigSyncPeriod)
		endpointSliceConfig.RegisterEventHandler(s.Proxier)
		go endpointSliceConfig.Run(wait.NeverStop)
	}

	if destinationRuleHandler, ok := s.Proxier.(config.DestinationRuleHandler); ok {
		destinationRuleConfig := config.NewDestinationRuleConfig(
			istioInformerFactory.Networking().V1alpha3().DestinationRules(), s.ConfigSyncPeriod)
		destinationRuleConfig.RegisterEventHandler(destinationRuleHandler)
		go destinationRuleConfig.Run(wait.NeverStop)
	}

	// This has to start after the calls to NewServiceConfig and NewEndpointsConfig because those
	// functions must configure their shared informer event handlers first.
	informerFactory.Start(wait.NeverStop)
	istioInformerFactory.Start(wait.NeverStop)

	go s.Proxier.SyncLoop()

	return nil
}

// CleanupAndExit remove iptables rules and ipset/ipvs rules in ipvs proxy mode
// and exit if success return nil
func (s *ProxyServer) CleanupAndExit() error {
	// cleanup IPv6 and IPv4 iptables rules
	ipts := []utiliptables.Interface{
		utiliptables.New(s.execer, utiliptables.ProtocolIPv4),
		utiliptables.New(s.execer, utiliptables.ProtocolIPv6),
	}
	var encounteredError bool
	for _, ipt := range ipts {
		encounteredError = userspace.CleanupLeftovers(ipt) || encounteredError
		//encounteredError = iptables.CleanupLeftovers(ipt) || encounteredError
		//encounteredError = ipvs.CleanupLeftovers(s.IpvsInterface, ipt, s.IpsetInterface) || encounteredError
	}
	if encounteredError {
		return errors.New("encountered an error while tearing down rules")
	}

	return nil
}
