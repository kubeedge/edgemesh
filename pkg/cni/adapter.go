package cni

import (
	"context"
	"fmt"
	"net"
	"os"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	utiliptables "k8s.io/kubernetes/pkg/util/iptables"
	"k8s.io/utils/exec"

	"github.com/kubeedge/edgemesh/pkg/apis/config/defaults"
	"github.com/kubeedge/edgemesh/pkg/apis/config/v1alpha1"
	"github.com/kubeedge/edgemesh/pkg/tunnel"
	"github.com/kubeedge/edgemesh/pkg/util/tunutils"
)

type Adapter interface {
	// HandleReceive deal with data from Pod to Tunnel
	TunToTunnel()

	// WatchRoute watch CIDR in overlayNetwork and insert Route to Tun dev
	WatchRoute() error

	// CloseRoute close all the Tun and stream
	CloseRoute()
}

var _ Adapter = (*MeshAdapter)(nil)

type MeshAdapter struct {
	kubeClient       clientset.Interface
	IptInterface     utiliptables.Interface
	execer           exec.Interface
	ConfigSyncPeriod time.Duration
	TunConn          *cni.TunConn
	HostCIDR         string
	Cloud            []string      // PodCIDR in cloud
	Edge             []string      // PodCIDR in edge
	Close            chan struct{} // stop signal
}

func NewMeshAdapter(cfg *v1alpha1.EdgeCNIConfig, cli clientset.Interface) (*MeshAdapter, error) {
	// get pod network info from cfg and APIServer
	cloud, edge, err := getCIDR(cfg.MeshCIDRConfig)
	if err != nil {
		klog.Errorf("get CIDR from config failed: %v", err)
		return nil, err
	}
	local, err := findLocalCIDR(cli)
	if err != nil {
		klog.Errorf("get localCIDR from apiserver failed: %v", err)
		return nil, err
	}

	// Create a iptables utils.
	execer := exec.New()
	iptIf := utiliptables.New(execer, utiliptables.ProtocolIPv4)

	// create a tun Connection stream
	tun, err := cni.NewTunConn(defaults.TunDeviceName)
	if err != nil {
		klog.Errorf("create tun device err: ", err)
		return nil, err
	}
	// setup tun and check it from ifconfig or ip link
	err = cni.SetupTunDevice(defaults.TunDeviceName)
	if err != nil {
		klog.Errorf("Setup Tun device failed:", err)
		return nil, err
	}

	return &MeshAdapter{
		kubeClient:   cli,
		IptInterface: iptIf,
		TunConn:      tun,
		HostCIDR:     local,
		Edge:         edge,
		Cloud:        cloud,
	}, nil
}

// getCIDR read from config file and get edge/cloud cidr user set
func getCIDR(cfg *v1alpha1.MeshCIDRConfig) ([]string, []string, error) {
	cloud := cfg.CloudCIDR
	edge := cfg.EdgeCIDR

	if err := validateCIDRs(cloud); err != nil {
		klog.ErrorS(err, "Cloud CIDR is not valid", "cidr", cloud)
		return nil, nil, err
	}

	if err := validateCIDRs(edge); err != nil {
		klog.ErrorS(err, "Edge CIDR is not valid", "cidr", edge)
		return nil, nil, err
	}

	klog.Infof("Parsed CIDR of Cloud: %v \n   Edge: %v \n", cloud, edge)
	return cloud, edge, nil
}

// check if the address validate
func validateCIDRs(cidrs []string) error {
	for _, cidr := range cidrs {
		if _, _, err := net.ParseCIDR(cidr); err != nil {
			return fmt.Errorf("invalid CIDR %s: %w", cidr, err)
		}
	}
	return nil
}

// get Local Pod CIDR
func findLocalCIDR(cli clientset.Interface) (string, error) {
	nodeName := os.Getenv("NODE_NAME")
	if nodeName == "" {
		klog.Errorf("NODE_NAME environment variable not set")
		return "", fmt.Errorf("the env NODE_NAME is not set")
	}

	// use clientset to get local info
	node, err := cli.CoreV1().Nodes().Get(context.Background(), nodeName, metav1.GetOptions{})
	if err != nil {
		klog.Errorf("get Node info from Apiserver failed:", err)
		return "", fmt.Errorf("failed to get Node: %w", err)
	}
	podCIDR := node.Spec.PodCIDR
	return podCIDR, nil
}

// CheckTunCIDR  if the cidr is not  in the same network
func (mesh *MeshAdapter) CheckTunCIDR(outerCidr string) (bool, error) {
	outerIP, outerNet, err := net.ParseCIDR(outerCidr)
	if err != nil {
		klog.Error("failed to parse outerCIDR: %v", err)
		return false, err
	}
	_, hostNet, err := net.ParseCIDR(mesh.HostCIDR)
	if err != nil {
		klog.Error("failed to parse hostCIDR: %v", err)
		return false, err
	}
	return hostNet.Contains(outerIP) && hostNet.Mask.String() == outerNet.Mask.String(), nil
}

func (mesh *MeshAdapter) Run() {
	klog.Infof("[CNI] Start Meshadapter")
	// get data from receive pipeline
	go mesh.TunToTunnel()
}

func (mesh *MeshAdapter) WatchRoute() error {
	// insert basic route to Tundev
	allCIDR := append(mesh.Edge, mesh.Cloud...)
	for _, cidr := range allCIDR {
		sameNet, err := mesh.CheckTunCIDR(cidr)
		if err != nil {
			klog.Errorf("Check if PodCIDR cross the  subnet failed:", err)
			return err
		}
		if !sameNet {
			err = cni.AddRouteToTun(cidr, defaults.TunDeviceName)
			if err != nil {
				klog.Errorf("\n Add route to TunDev failed: ", err)
				continue
			}
		}
	}
	// Insert IPtable rule to make sure Other CNIs do not make SNAT
	rule, err := mesh.IptInterface.EnsureRule("-I", "nat", "POSTROUTING", "1", "-s", mesh.HostCIDR, "!", "-o", "docker0", "-j", "ACCEPT")
	if err != nil {
		klog.Errorf("Insert iptable rule :%s failed", rule, err)
		return err
	}
	klog.Infof("Insert iptable rule :%s", rule)
	return nil
	// TODO： watch the subNetwork event and if the cidr changes ,apply that change to node
}

func (mesh *MeshAdapter) TunToTunnel() {
	// Listen at TunDev and Receive data to TunConn Buffer
	go mesh.TunConn.TunReceiveLoop()
	go mesh.HandleReceiveFromTun()
}

func (mesh *MeshAdapter) HandleReceiveFromTun() {
	buffer := cni.NewRecycleByteBuffer(65536)
	tun := mesh.TunConn
	for {
		select {
		case <-mesh.Close:
			klog.Infof("Close HandleReceive Process")
			return
		case packet := <-tun.ReceivePipe:
			//set CNI Options
			n := len(packet)
			buffer.Write(packet[:n])
			frame, err := cni.ParseIPFrame(buffer)
			NodeName, err := mesh.GetNodeNameByPodIP(frame.GetTargetIP())
			if err != nil {
				klog.Errorf("get NodeName by PodIP failed")
			}
			cniOpts := tunnel.ProxyOptions{
				Protocol: frame.GetProtocol(),
				NodeName: NodeName,
			}
			stream, err := tunnel.Agent.GetCNIAdapterStream(cniOpts)
			if err != nil {
				klog.Errorf("l3 adapter get proxy stream from %s error: %w", cniOpts.NodeName, err)
				return
			}
			_, err = stream.Write(frame.ToBytes())
			if err != nil {
				klog.Errorf("Error writing data: %v\n", err)
				return
			}
			klog.Infof("send Data to %s", frame.GetTargetIP())
			klog.Infof("l3 adapter start proxy data between nodes %v", cniOpts.NodeName)
			klog.Infof("Success proxy to %v", tun)
		default:
			continue
		}
	}
}

func (mesh *MeshAdapter) GetNodeNameByPodIP(podIP string) (string, error) {
	// use FieldSelector to get Pod Name
	pods, err := mesh.kubeClient.CoreV1().Pods("").List(context.Background(), metav1.ListOptions{
		FieldSelector: "status.podIP=" + podIP,
	})
	if err != nil {
		klog.Errorf("Error getting pods: %v", err)
		return "", err
	}

	if len(pods.Items) == 0 {
		klog.Errorf("No pod found with IP: %s", podIP)
		return "", err
	}

	return pods.Items[0].Spec.NodeName, nil
}

func (mesh *MeshAdapter) CloseRoute() {
	close(mesh.Close)
	err := mesh.TunConn.CleanTunRoute(defaults.TunDeviceName)
	if err != nil {
		klog.Info("Clean Route failed")
	}
	err = mesh.TunConn.CleanTunDevice()
	if err != nil {
		klog.Info("Clean Tun Dev failed")
	}
}
