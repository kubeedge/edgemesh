package tunnel

import (
	"math"

	"github.com/libp2p/go-libp2p"
	rcmgr "github.com/libp2p/go-libp2p/p2p/host/resource-manager"

	"github.com/kubeedge/edgemesh/pkg/apis/config/defaults"
	"github.com/kubeedge/edgemesh/pkg/apis/config/v1alpha1"
)

func useLimit(config *v1alpha1.TunnelLimitConfig) *rcmgr.ScalingLimitConfig {
	scalingLimits := rcmgr.DefaultLimits
	protoLimit := rcmgr.BaseLimit{
		Streams:         config.TunnelBaseStreamIn + config.TunnelBaseStreamOut,
		StreamsInbound:  config.TunnelBaseStreamIn,
		StreamsOutbound: config.TunnelBaseStreamOut,
		FD:              rcmgr.DefaultLimits.ProtocolBaseLimit.FD,
		Memory:          rcmgr.DefaultLimits.ProtocolBaseLimit.Memory,
	}
	scalingLimits.AddProtocolLimit(defaults.ProxyProtocol, protoLimit, rcmgr.DefaultLimits.ProtocolLimitIncrease)
	scalingLimits.ProtocolPeerBaseLimit.Streams = config.TunnelPeerBaseStreamIn + config.TunnelPeerBaseStreamOut
	scalingLimits.ProtocolPeerBaseLimit.StreamsOutbound = config.TunnelPeerBaseStreamOut
	scalingLimits.ProtocolPeerBaseLimit.StreamsInbound = config.TunnelPeerBaseStreamIn
	return &scalingLimits
}
func useNoLimit() *rcmgr.ScalingLimitConfig {
	scalingLimits := rcmgr.DefaultLimits
	protoLimit := rcmgr.BaseLimit{
		Streams:         math.MaxInt,
		StreamsInbound:  math.MaxInt,
		StreamsOutbound: math.MaxInt,
		FD:              rcmgr.DefaultLimits.ProtocolBaseLimit.FD,
		Memory:          rcmgr.DefaultLimits.ProtocolBaseLimit.Memory,
	}
	scalingLimits.AddProtocolLimit(defaults.ProxyProtocol, protoLimit, rcmgr.BaseLimitIncrease{})
	scalingLimits.ProtocolPeerBaseLimit.Streams = math.MaxInt
	scalingLimits.ProtocolPeerBaseLimit.StreamsOutbound = math.MaxInt
	scalingLimits.ProtocolPeerBaseLimit.StreamsInbound = math.MaxInt
	scalingLimits.ProtocolPeerLimitIncrease = rcmgr.BaseLimitIncrease{}
	return &scalingLimits
}

func buildLimitOpt(scalingLimits *rcmgr.ScalingLimitConfig) (libp2p.Option, error) {
	// Add limits around included libp2p protocols
	libp2p.SetDefaultServiceLimits(scalingLimits)
	// Turn the scaling limits into a static set of limits using `.AutoScale`. This
	// scales the limits proportional to your system memory.
	limits := scalingLimits.AutoScale()
	// The resource manager expects a limiter, se we create one from our limits.
	limiter := rcmgr.NewFixedLimiter(limits)
	// Initialize the resource manager
	if rm, err := rcmgr.NewResourceManager(limiter); err != nil {
		return nil, err
	} else {
		return libp2p.ResourceManager(rm), nil
	}
}

func CreateLimitOpt(config *v1alpha1.TunnelLimitConfig) (libp2p.Option, error) {
	// Start with the default scaling limits.
	var scaleLimit *rcmgr.ScalingLimitConfig
	// Adjust limit if need:
	if !config.Enable {
		scaleLimit = useNoLimit()
	} else {
		scaleLimit = useLimit(config)
	}
	return buildLimitOpt(scaleLimit)
}
