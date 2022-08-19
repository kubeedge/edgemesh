package connmgr

import (
	"time"

	lconnmgr "github.com/libp2p/go-libp2p/p2p/net/connmgr"
)

// Option represents an option for the basic connection manager.
// Deprecated: use go-libp2p/p2p/net/connmgr.Option instead.
type Option = lconnmgr.Option

// DecayerConfig applies a configuration for the decayer.
// Deprecated: use go-libp2p/p2p/net/connmgr.DecayerConfig instead.
func DecayerConfig(opts *DecayerCfg) Option {
	return lconnmgr.DecayerConfig(opts)
}

// WithGracePeriod sets the grace period.
// The grace period is the time a newly opened connection is given before it becomes
// subject to pruning.
// Deprecated: use go-libp2p/p2p/net/connmgr.WithGracePeriod instead.
func WithGracePeriod(p time.Duration) Option {
	return lconnmgr.WithGracePeriod(p)
}

// WithSilencePeriod sets the silence period.
// The connection manager will perform a cleanup once per silence period
// if the number of connections surpasses the high watermark.
// Deprecated: use go-libp2p/p2p/net/connmgr.WithSilencePeriod instead.
func WithSilencePeriod(p time.Duration) Option {
	return lconnmgr.WithSilencePeriod(p)
}
