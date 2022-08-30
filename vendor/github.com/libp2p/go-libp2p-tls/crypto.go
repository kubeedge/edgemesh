package libp2ptls

import (
	libp2ptls "github.com/libp2p/go-libp2p/p2p/security/tls"

	ic "github.com/libp2p/go-libp2p-core/crypto"
)

// Identity is used to secure connections
// Deprecated: use github.com/libp2p/go-libp2p/p2p/security/tls.Identity instead.
type Identity = libp2ptls.Identity

// NewIdentity creates a new identity
// Deprecated: use github.com/libp2p/go-libp2p/p2p/security/tls.NewIdentity instead.
func NewIdentity(privKey ic.PrivKey) (*Identity, error) {
	return libp2ptls.NewIdentity(privKey)
}
