package routinghelpers

import (
	"context"
	"time"

	"github.com/libp2p/go-libp2p/core/routing"
	"github.com/multiformats/go-multihash"
)

type ParallelRouter struct {
	Timeout      time.Duration
	IgnoreError  bool
	Router       routing.Routing
	ExecuteAfter time.Duration
}

type SequentialRouter struct {
	Timeout     time.Duration
	IgnoreError bool
	Router      routing.Routing
}

type ProvideManyRouter interface {
	ProvideMany(ctx context.Context, keys []multihash.Multihash) error
	Ready() bool
}
