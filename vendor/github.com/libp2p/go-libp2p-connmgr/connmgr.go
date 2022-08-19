// Deprecated: This package has moved into go-libp2p as a sub-package: github.com/libp2p/go-libp2p/p2p/net/connmgr.
package connmgr

import (
	lconnmgr "github.com/libp2p/go-libp2p/p2p/net/connmgr"
)

// BasicConnMgr is a ConnManager that trims connections whenever the count exceeds the
// high watermark. New connections are given a grace period before they're subject
// to trimming. Trims are automatically run on demand, only if the time from the
// previous trim is higher than 10 seconds. Furthermore, trims can be explicitly
// requested through the public interface of this struct (see TrimOpenConns).
//
// See configuration parameters in NewConnManager.
// Deprecated: use go-libp2p/p2p/net/connmgr.BasicConnMgr instead.
type BasicConnMgr = lconnmgr.BasicConnMgr

// NewConnManager creates a new BasicConnMgr with the provided params:
// lo and hi are watermarks governing the number of connections that'll be maintained.
// When the peer count exceeds the 'high watermark', as many peers will be pruned (and
// their connections terminated) until 'low watermark' peers remain.
// Deprecated: use go-libp2p/p2p/net/connmgr.NewConnManager instead.
func NewConnManager(low, hi int, opts ...lconnmgr.Option) (*BasicConnMgr, error) {
	return lconnmgr.NewConnManager(low, hi, opts...)
}
