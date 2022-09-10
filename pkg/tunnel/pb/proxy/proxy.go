package proxy

import "github.com/libp2p/go-libp2p/core/protocol"

// ProxyProtocol maintained by EdgeMesh Author.
const ProxyProtocol protocol.ID = "/libp2p/tunnel-proxy/1.0.0"

type ProxyOptions struct {
	Protocol string
	NodeName string
	IP       string
	Port     int32
}
