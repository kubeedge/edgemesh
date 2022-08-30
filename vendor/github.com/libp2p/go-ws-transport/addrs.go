package websocket

import (
	"net"

	"github.com/libp2p/go-libp2p/p2p/transport/websocket"

	ma "github.com/multiformats/go-multiaddr"
)

// Addr is an implementation of net.Addr for WebSocket.
// Deprecated: use github.com/libp2p/go-libp2p/p2p/transport/websocket.Addr instead.
type Addr = websocket.Addr

// NewAddr creates an Addr with `ws` scheme (insecure).
//
// Deprecated. Use NewAddrWithScheme.
func NewAddr(host string) *Addr {
	// Older versions of the transport only supported insecure connections (i.e.
	// WS instead of WSS). Assume that is the case here.
	return NewAddrWithScheme(host, false)
}

// NewAddrWithScheme creates a new Addr using the given host string. isSecure
// should be true for WSS connections and false for WS.
// Deprecated: use github.com/libp2p/go-libp2p/p2p/transport/websocket.NewAddrWithScheme instead.
func NewAddrWithScheme(host string, isSecure bool) *Addr {
	return websocket.NewAddrWithScheme(host, isSecure)
}

// Deprecated: use github.com/libp2p/go-libp2p/p2p/transport/websocket.ConvertWebsocketMultiaddrToNetAddr instead.
func ConvertWebsocketMultiaddrToNetAddr(maddr ma.Multiaddr) (net.Addr, error) {
	return websocket.ConvertWebsocketMultiaddrToNetAddr(maddr)
}

// Deprecated: use github.com/libp2p/go-libp2p/p2p/transport/websocket.ParseWebsocketNetAddr instead.
func ParseWebsocketNetAddr(a net.Addr) (ma.Multiaddr, error) {
	return websocket.ParseWebsocketNetAddr(a)
}
