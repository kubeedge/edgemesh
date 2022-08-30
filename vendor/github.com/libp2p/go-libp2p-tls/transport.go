// Deprecated: This package has moved into go-libp2p as a sub-package: github.com/libp2p/go-libp2p/p2p/security/tls.
package libp2ptls

import (
	libp2ptls "github.com/libp2p/go-libp2p/p2p/security/tls"

	ci "github.com/libp2p/go-libp2p-core/crypto"
)

// ID is the protocol ID (used when negotiating with multistream)
// Deprecated: use github.com/libp2p/go-libp2p/p2p/security/tls.ID instead.
const ID = "/tls/1.0.0"

// Transport constructs secure communication sessions for a peer.
// Deprecated: use github.com/libp2p/go-libp2p/p2p/security/tls.Transport instead.
type Transport = libp2ptls.Transport

// New creates a TLS encrypted transport
// Deprecated: use github.com/libp2p/go-libp2p/p2p/security/tls.New instead.
func New(key ci.PrivKey) (*Transport, error) {
	return libp2ptls.New(key)
}
