package util

import (
	"fmt"

	"github.com/libp2p/go-libp2p"
	quic "github.com/libp2p/go-libp2p-quic-transport"
	ws "github.com/libp2p/go-ws-transport"
)

// GenerateTransportOption generate transport option from protocol
// supported protocols: ["tcp", "ws", "quic"]
func GenerateTransportOption(protocol string) libp2p.Option {
	var opt libp2p.Option
	switch protocol {
	case "tcp":
		// libp2p.Transport(tcp.NewTCPTransport) is enable by default
	case "ws":
		opt = libp2p.Transport(ws.New)
	case "quic":
		opt = libp2p.Transport(quic.NewTransport)
	}
	return opt
}

// GenerateMultiAddr generate an IPv4 multi-address from protocol, ip and port
// supported protocols: ["tcp", "ws", "quic"]
func GenerateMultiAddr(protocol, ip string, port int) string {
	var multiAddr string
	switch protocol {
	case "tcp":
		multiAddr = fmt.Sprintf("/ip4/%s/tcp/%d", ip, port)
	case "ws":
		multiAddr = fmt.Sprintf("/ip4/%s/tcp/%d/ws", ip, port)
	case "quic":
		multiAddr = fmt.Sprintf("/ip4/%s/udp/%d/quic", ip, port)
	}
	return multiAddr
}
