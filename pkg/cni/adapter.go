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
		return nil, fmt.Errorf("failed to get CIDR from config, error: %v", err)
	}
	klog.Infof("the cloud CIDRs are %v, the edge CIDRs are %v", cloud, edge)

	local, err := findLocalCIDR(cli)
	if err != nil {
		return nil, fmt.Errorf("failed to get local CIDR from apiserver, error: %v", err)
	}
	klog.Infof("local CIDR is %v", local)

	// Create a iptables utils.
	execer := exec.New()
	iptIf := utiliptables.New(execer, utiliptables.ProtocolIPv4)

	// create a tun Connection stream
	tun, err := cni.NewTunConn(defaults.TunDeviceName)
	if err != nil {
		return nil, fmt.Errorf("failed to create tun device %s, error: %v", defaults.TunDeviceName, err)
	}
	// setup tun and check it from ifconfig or ip link
	err = cni.SetupTunDevice(defaults.TunDeviceName)
	if err != nil {
		return nil, fmt.Errorf("failed to set up tun device %s, error: %v", defaults.TunDeviceName, err)
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
		return nil, nil, fmt.Errorf("cloud CIDRs are invalid, error: %v", err)
	}

	if err := validateCIDRs(edge); err != nil {
		return nil, nil, fmt.Errorf("edge CIDRs are invalid, error: %v", err)
	}

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
		return "", fmt.Errorf("the env NODE_NAME is not set")
	}

	// use clientset to get local info
	node, err := cli.CoreV1().Nodes().Get(context.Background(), nodeName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get Node %s, error: %v", nodeName, err)
	}
	podCIDR := node.Spec.PodCIDR
	return podCIDR, nil
}

// CheckTunCIDR check whether the mesh CIDR and the given parameter CIDR are in the same network or not.
func (mesh *MeshAdapter) CheckTunCIDR(outerCidr string) (bool, error) {
	outerIP, outerNet, err := net.ParseCIDR(outerCidr)
	if err != nil {
		return false, fmt.Errorf("failed to parse outerCIDR %s, error:%v", outerCidr, err)
	}

	_, hostNet, err := net.ParseCIDR(mesh.HostCIDR)
	if err != nil {
		return false, fmt.Errorf("failed to parse hostCIDR %s, error: %v", mesh.HostCIDR, err)
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
			return fmt.Errorf("failed to check whether CIDRs are in the same network or not, error: %v", err)
		}
		if !sameNet {
			err = cni.AddRouteToTun(cidr, defaults.TunDeviceName)
			if err != nil {
				klog.Errorf("failed to add route to TunDev, error: %v", err)
				continue
			}
		}
	}
	// Insert IPtable rule to make sure Other CNIs do not make SNAT
	rule, err := mesh.IptInterface.EnsureRule("-I", "nat", "POSTROUTING", "1", "-s", mesh.HostCIDR, "!", "-o", "docker0", "-j", "ACCEPT")
	if err != nil {
		return fmt.Errorf("failed to insert iptable rule, error: %v", err)
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
	buffer := cni.NewRecycleByteBuffer(cni.PacketSize)
	tun := mesh.TunConn
	for {
		select {
		case <-mesh.Close:
			klog.Warningln("Close HandleReceive Process")
			return
		case packet := <-tun.ReceivePipe:
			//set CNI Options
			n := len(packet)
			buffer.Write(packet[:n])
			frame, err := cni.ParseIPFrame(buffer)
			if err != nil {
				klog.Errorf("failed to parse IP frame, error: %v", err)
				continue
			}

			nodeName, err := mesh.GetNodeNameByPodIP(frame.GetTargetIP())
			if err != nil {
				klog.Errorf("failed to get NodeName by PodIP %s, error: %v", frame.GetTargetIP(), err)
				continue
			}
			klog.Infof("find node %s by Pod IP %s", nodeName, frame.GetTargetIP())

			cniOpts := tunnel.ProxyOptions{
				Protocol: frame.GetProtocol(),
				NodeName: nodeName,
			}
			stream, err := tunnel.Agent.GetCNIAdapterStream(cniOpts)
			if err != nil {
				klog.Errorf("l3 adapter failed to get proxy stream from %s, error: %v", cniOpts.NodeName, err)
				continue
			}
			_, err = stream.Write(frame.ToBytes())
			if err != nil {
				klog.Errorf("failed to write stream data, error: %v", err)
				continue
			}
			klog.Infof("send Data to %s", frame.GetTargetIP())
			klog.Infof("l3 adapter start proxy data between nodes %v", cniOpts.NodeName)
			klog.Infof("Success proxy to %v", tun)
		}
	}
}

func (mesh *MeshAdapter) GetNodeNameByPodIP(podIP string) (string, error) {
	// use FieldSelector to get Pod Name
	pods, err := mesh.kubeClient.CoreV1().Pods("").List(context.Background(), metav1.ListOptions{
		FieldSelector: "status.podIP=" + podIP,
	})
	if err != nil {
		return "", err
	}

	if len(pods.Items) == 0 {
		return "", fmt.Errorf("no pod found with IP %s", podIP)
	}

	return pods.Items[0].Spec.NodeName, nil
}

func (mesh *MeshAdapter) CloseRoute() {
	close(mesh.Close)
	err := mesh.TunConn.CleanTunRoute()
	if err != nil {
		klog.Errorf("failed to clean tun route, error: %v", err)
	}
	err = mesh.TunConn.CleanTunDevice()
	if err != nil {
		klog.Errorf("failed to clean tun device, error: %v", err)
	}
}
