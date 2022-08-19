//go:build cgo
// +build cgo

package connmgr

import (
	lconnmgr "github.com/libp2p/go-libp2p/p2p/net/connmgr"
)

// WithEmergencyTrim is an option to enable trimming connections on memory emergency.
// Deprecated: use go-libp2p/p2p/net/connmgr.WithEmergencyTrim instead.
func WithEmergencyTrim(enable bool) Option {
	return lconnmgr.WithEmergencyTrim(enable)
}
