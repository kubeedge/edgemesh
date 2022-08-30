package tcp

import (
	"github.com/libp2p/go-libp2p/p2p/transport/tcp"
)

// ReuseportIsAvailable returns whether reuseport is available to be used. This
// is here because we want to be able to turn reuseport on and off selectively.
// For now we use an ENV variable, as this handles our pressing need:
//
//   LIBP2P_TCP_REUSEPORT=false ipfs daemon
//
// If this becomes a sought after feature, we could add this to the config.
// In the end, reuseport is a stop-gap.
//
// Deprecated: use github.com/libp2p/go-libp2p/p2p/transport/tcp.ReuseportIsAvailable instead.
func ReuseportIsAvailable() bool {
	return tcp.ReuseportIsAvailable()
}
