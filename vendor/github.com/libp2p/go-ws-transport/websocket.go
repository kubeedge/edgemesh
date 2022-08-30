// Package websocket implements a websocket based transport for go-libp2p.
// Deprecated: This package has moved into go-libp2p as a sub-package: github.com/libp2p/go-libp2p/p2p/transport/websocket.
package websocket

import (
	"crypto/tls"

	"github.com/libp2p/go-libp2p/p2p/transport/websocket"

	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/transport"
)

// WsFmt is multiaddr formatter for WsProtocol
// Deprecated: use github.com/libp2p/go-libp2p/p2p/transport/websocket.WsFmt instead.
var WsFmt = websocket.WsFmt

// Deprecated: use github.com/libp2p/go-libp2p/p2p/transport/websocket.Option instead.
type Option = websocket.Option

// WithTLSClientConfig sets a TLS client configuration on the WebSocket Dialer. Only
// relevant for non-browser usages.
//
// Some useful use cases include setting InsecureSkipVerify to `true`, or
// setting user-defined trusted CA certificates.
// Deprecated: use github.com/libp2p/go-libp2p/p2p/transport/websocket.WithTLSClientConfig instead.
func WithTLSClientConfig(c *tls.Config) Option {
	return websocket.WithTLSClientConfig(c)
}

// WithTLSConfig sets a TLS configuration for the WebSocket listener.
// Deprecated: use github.com/libp2p/go-libp2p/p2p/transport/websocket.WithTLSConfig instead.
func WithTLSConfig(conf *tls.Config) Option {
	return websocket.WithTLSConfig(conf)
}

// WebsocketTransport is the actual go-libp2p transport
// Deprecated: use github.com/libp2p/go-libp2p/p2p/transport/websocket.WebsocketTransport instead.
type WebsocketTransport = websocket.WebsocketTransport

// Deprecated: use github.com/libp2p/go-libp2p/p2p/transport/websocket.New instead.
func New(u transport.Upgrader, rcmgr network.ResourceManager, opts ...Option) (*WebsocketTransport, error) {
	return websocket.New(u, rcmgr, opts...)
}
