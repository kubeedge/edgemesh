package gateway

import (
	"fmt"
	"net"
	"sync"
	"time"

	istio "istio.io/client-go/pkg/clientset/versioned"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/edgemesh/pkg/apis/config/defaults"
	"github.com/kubeedge/edgemesh/pkg/apis/config/v1alpha1"
	"github.com/kubeedge/edgemesh/pkg/clients"
	"github.com/kubeedge/edgemesh/pkg/loadbalancer"
)

// EdgeGateway is an edge ingress gateway
type EdgeGateway struct {
	Config           *v1alpha1.EdgeGatewayConfig
	listenIPs        []net.IP
	lock             sync.Mutex           // protect serversByGateway
	serversByGateway map[string][]*Server // gatewayNamespace.gatewayName --> servers
	kubeClient       kubernetes.Interface
	istioClient      istio.Interface
	loadBalancer     *loadbalancer.LoadBalancer
	syncPeriod       time.Duration
	stopCh           chan struct{}
}

// Name of edgegateway
func (gw *EdgeGateway) Name() string {
	return defaults.EdgeGatewayModuleName
}

// Group of edgegateway
func (gw *EdgeGateway) Group() string {
	return defaults.EdgeGatewayModuleName
}

// Enable indicates whether enable this module
func (gw *EdgeGateway) Enable() bool {
	return gw.Config.Enable
}

// Start edgegateway
func (gw *EdgeGateway) Start() {
	err := gw.Run()
	if err != nil {
		klog.Errorf("Failed to run EdgeGateway error: %v", err)
		return
	}

	<-beehiveContext.Done()
}

// Shutdown edgegateway
func (gw *EdgeGateway) Shutdown() {
	// TODO graceful shutdown
}

// Register edgegateway to beehive modules
func Register(c *v1alpha1.EdgeGatewayConfig, cli *clients.Clients) error {
	gw, err := newEdgeGateway(c, cli)
	if err != nil {
		return fmt.Errorf("register module %s error: %v", defaults.EdgeGatewayModuleName, err)
	}
	core.Register(gw)
	return nil
}

func newEdgeGateway(c *v1alpha1.EdgeGatewayConfig, cli *clients.Clients) (*EdgeGateway, error) {
	if !c.Enable {
		return &EdgeGateway{Config: c}, nil
	}

	klog.V(4).Infof("Start get ips which need listen...")
	listenIPs, err := GetIPsNeedListen(c)
	if err != nil {
		return nil, fmt.Errorf("get GetIPsNeedListen err: %w", err)
	}
	klog.Infof("Gateway listen ips: %+v", listenIPs)

	kubeClient := cli.GetKubeClient()
	istioClient := cli.GetIstioClient()
	syncPeriod := 15 * time.Minute // TODO get from config
	loadBalancer := loadbalancer.New(c.LoadBalancer, kubeClient, istioClient, syncPeriod)
	initLoadBalancer(loadBalancer)

	return &EdgeGateway{
		Config:           c,
		listenIPs:        listenIPs,
		serversByGateway: make(map[string][]*Server),
		kubeClient:       kubeClient,
		istioClient:      istioClient,
		loadBalancer:     loadBalancer,
		syncPeriod:       syncPeriod,
		stopCh:           make(chan struct{}),
	}, nil
}
