// Deprecated: This package has moved into go-libp2p as a sub-package: github.com/libp2p/go-libp2p/p2p/transport/tcp.
package tcp

import (
	"time"

	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/transport"

	"github.com/libp2p/go-libp2p/p2p/transport/tcp"
)

// Deprecated: use github.com/libp2p/go-libp2p/p2p/transport/tcp.Option instead.
type Option = tcp.Option

// Deprecated: use github.com/libp2p/go-libp2p/p2p/transport/tcp.DisableReuseport instead.
func DisableReuseport() Option {
	return tcp.DisableReuseport()
}

// Deprecated: use github.com/libp2p/go-libp2p/p2p/transport/tcp.WithConnectionTimeout instead.
func WithConnectionTimeout(d time.Duration) Option {
	return tcp.WithConnectionTimeout(d)
}

// TcpTransport is the TCP transport.
// Deprecated: use github.com/libp2p/go-libp2p/p2p/transport/tcp.TcpTransport instead.
type TcpTransport = tcp.TcpTransport

// NewTCPTransport creates a tcp transport object that tracks dialers and listeners
// created. It represents an entire TCP stack (though it might not necessarily be).
// Deprecated: use github.com/libp2p/go-libp2p/p2p/transport/tcp.NewTCPTransport instead.
func NewTCPTransport(upgrader transport.Upgrader, rcmgr network.ResourceManager, opts ...Option) (*TcpTransport, error) {
	return tcp.NewTCPTransport(upgrader, rcmgr, opts...)
}
