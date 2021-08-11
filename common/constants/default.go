package constants

// Resources
const (
	// Certificates
	ServerDefaultACLDirectory = "/etc/kubeedge/edgemesh/server/acls"
	AgentDefaultACLDirectory  = "/etc/kubeedge/edgemesh/agent/acls"
	ServerDefaultKeyFile      = ServerDefaultACLDirectory + "/server.key"
	AgentDefaultKeyFile       = AgentDefaultACLDirectory + "/server.key"

	SecretNamespace  = "kubeedge"
	SecretName       = "edgemeshaddrsecret"
	ServerAddrName   = "edgemeshserver"
	ConnectionClosed = "use of closed network connection"
	StreamReset      = "stream reset"

	// env
	MY_NODE_NAME     = "MY_NODE_NAME"
)
