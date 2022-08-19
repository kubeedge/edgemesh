package connmgr

import (
	"github.com/libp2p/go-libp2p-core/connmgr"
	lconnmgr "github.com/libp2p/go-libp2p/p2p/net/connmgr"
)

// DecayerCfg is the configuration object for the Decayer.
// Deprecated: use go-libp2p/p2p/net/connmgr.DecayerCfg instead.
type DecayerCfg = lconnmgr.DecayerCfg

// NewDecayer creates a new decaying tag registry.
// Deprecated: use go-libp2p/p2p/net/connmgr.NewDecayer instead.
func NewDecayer(cfg *DecayerCfg, mgr *BasicConnMgr) (connmgr.Decayer, error) {
	return lconnmgr.NewDecayer(cfg, mgr)
}
