package discovery

import "github.com/libp2p/go-libp2p-core/protocol"

const DiscoveryProtocol protocol.ID = "/libp2p/tunnel-discovery/1.0.0"

type DiscoveryType string

const (
	MdnsDiscovery DiscoveryType = "MDNS"
	DhtDiscovery  DiscoveryType = "DHT"
)
