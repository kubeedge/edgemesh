package defaults

type TunnelMode string

const (
	ConfigDir               = "/etc/edgemesh/config/"
	EdgeMeshAgentConfigName = "edgemesh-agent.yaml"

	EdgeDNSModuleName     = "EdgeDNS"
	EdgeProxyModuleName   = "EdgeProxy"
	EdgeGatewayModuleName = "EdgeGateway"
	EdgeTunnelModuleName  = "EdgeTunnel"

	DummyDeviceName   = "edgemesh0"
	DummyDeviceIP     = "169.254.96.16"          // TODO change dummy to bridge
	MetaServerAddress = "http://127.0.0.1:10550" // EdgeCore's metaServer address TODO get from config

	EdgeMode  = "EdgeMode"  // detected running on the edge
	CloudMode = "CloudMode" // detected running on the cloud
	DebugMode = "DebugMode" // detected that user manually configured kubeAPIConfig

	ClientMode       TunnelMode = "ClientOnly"
	ServerClientMode TunnelMode = "ServerAndClient"

	Rendezvous = "EDGEMESH_PLAYGOUND"
	PSKPath    = "/etc/edgemesh/psk"
)
