// Deprecated: This package has moved into go-libp2p as a sub-package: github.com/libp2p/go-libp2p/p2p/transport/quic.
package libp2pquic

import (
	libp2pquic "github.com/libp2p/go-libp2p/p2p/transport/quic"

	"github.com/libp2p/go-libp2p-core/connmgr"
	ic "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/pnet"
	tpt "github.com/libp2p/go-libp2p-core/transport"
)

// Deprecated: use github.com/libp2p/go-libp2p/p2p/transport/quic.ErrHolePunching instead.
var ErrHolePunching = libp2pquic.ErrHolePunching

// Deprecated: use github.com/libp2p/go-libp2p/p2p/transport/quic.HolePunchTimeout instead.
var HolePunchTimeout = libp2pquic.HolePunchTimeout

// NewTransport creates a new QUIC transport
// Deprecated: use github.com/libp2p/go-libp2p/p2p/transport/quic.NewTransport instead.
func NewTransport(key ic.PrivKey, psk pnet.PSK, gater connmgr.ConnectionGater, rcmgr network.ResourceManager) (tpt.Transport, error) {
	return libp2pquic.NewTransport(key, psk, gater, rcmgr)
}
