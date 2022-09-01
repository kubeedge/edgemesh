package defaults

const (
	ConfigDir               = "/etc/kubeedge/config/"
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
	DebugMode = "DebugMode" // detected that user manually configured kubeconfig
)

type TunnelMode string

const (
	ClientMode       TunnelMode = "ClientOnly"
	ServerClientMode TunnelMode = "ServerAndClient"
)
