package defaults

import (
	"github.com/libp2p/go-libp2p/core/protocol"
)

type TunnelMode string
type DiscoveryType string

const (
	ConfigDir                 = "/etc/edgemesh/config/"
	EdgeMeshAgentConfigName   = "edgemesh-agent.yaml"
	EdgeMeshGatewayConfigName = "edgemesh-gateway.yaml"

	EdgeDNSModuleName     = "EdgeDNS"
	EdgeProxyModuleName   = "EdgeProxy"
	EdgeGatewayModuleName = "EdgeGateway"
	EdgeTunnelModuleName  = "EdgeTunnel"

	BridgeDeviceName  = "edgemesh0"
	BridgeDeviceIP    = "169.254.96.16"
	MetaServerAddress = "http://127.0.0.1:10550"

	EdgeMode  = "EdgeMode"  // detected running on the edge
	CloudMode = "CloudMode" // detected running on the cloud
	DebugMode = "DebugMode" // detected that user manually configured kubeAPIConfig

	EmptyNodeName = "EMPTY_NODE_NAME"
	EmptyPodName  = "EMPTY_POD_NAME"

	// LabelEdgeMeshServiceProxyName indicates that an alternative service
	// proxy will implement this Service.
	LabelEdgeMeshServiceProxyName = "service.edgemesh.kubeedge.io/service-proxy-name"

	ProxyCaller   = "ProxyCaller"
	GatewayCaller = "GatewayCaller"

	ClientMode       TunnelMode = "ClientOnly"
	ServerClientMode TunnelMode = "ServerAndClient"

	Rendezvous = "EDGEMESH_PLAYGOUND"
	PSKPath    = "/etc/edgemesh/psk"

	// DiscoveryProtocol and ProxyProtocol maintained by EdgeMesh Author
	DiscoveryProtocol protocol.ID = "/libp2p/tunnel-discovery/1.0.0"
	ProxyProtocol     protocol.ID = "/libp2p/tunnel-proxy/1.0.0"

	MdnsDiscovery DiscoveryType = "MDNS"
	DhtDiscovery  DiscoveryType = "DHT"
)
