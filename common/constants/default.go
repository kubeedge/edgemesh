package constants

// Resources
const (
	ResourceTypeSecret = "secret"

	MY_NODE_NAME    = "MY_NODE_NAME"

	NamespaceSystem = "kubeedge"

	ServerDefaultACLDirectory = "/etc/kubeedge/edgemesh/server/acls"
	AgentDefaultACLDirectory = "/etc/kubeedge/edgemesh/agent/acls"
	ServerDefaultKeyFile    = ServerDefaultACLDirectory + "/server.key"
	AgentDefaultKeyFile     = AgentDefaultACLDirectory + "/server.key"

	SECRET_NAMESPACE = "kubeedge"
	SECRET_NAME      = "edgemeshaddrsecret"
	SERVER_ADDR_NAME = "edgemeshserver"
)
