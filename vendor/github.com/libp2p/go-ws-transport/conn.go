package websocket

import (
	"github.com/libp2p/go-libp2p/p2p/transport/websocket"

	ws "github.com/gorilla/websocket"
)

// GracefulCloseTimeout is the time to wait trying to gracefully close a
// connection before simply cutting it.
// Deprecated: use github.com/libp2p/go-libp2p/p2p/transport/websocket.GracefulCloseTimeout instead.
var GracefulCloseTimeout = websocket.GracefulCloseTimeout

// Conn implements net.Conn interface for gorilla/websocket.
// Deprecated: use github.com/libp2p/go-libp2p/p2p/transport/websocket.Conn instead.
type Conn = websocket.Conn

// NewConn creates a Conn given a regular gorilla/websocket Conn.
// Deprecated: use github.com/libp2p/go-libp2p/p2p/transport/websocket.NewConn instead.
func NewConn(raw *ws.Conn, secure bool) *Conn {
	return websocket.NewConn(raw, secure)
}
