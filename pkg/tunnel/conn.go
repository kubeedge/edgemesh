package tunnel

import (
	"net"

	"github.com/libp2p/go-libp2p/core/network"
)

// StreamAddr implements the net.Addr interface
type StreamAddr struct {
	protocol  string
	multiAddr string
}

func (sa *StreamAddr) Network() string {
	return sa.protocol
}

func (sa *StreamAddr) String() string {
	return sa.multiAddr
}

// StreamConn is libp2p network.Stream wrapper,
// which implements the Golang net.Conn interface
type StreamConn struct {
	network.Stream
	laddr *StreamAddr
	raddr *StreamAddr
}

func NewStreamConn(s network.Stream) *StreamConn {
	laddr := &StreamAddr{protocol: string(s.Protocol()), multiAddr: s.Conn().LocalMultiaddr().String()}
	raddr := &StreamAddr{protocol: string(s.Protocol()), multiAddr: s.Conn().RemoteMultiaddr().String()}
	return &StreamConn{
		Stream: s,
		laddr:  laddr,
		raddr:  raddr,
	}
}

// LocalAddr returns the local network address.
func (ns *StreamConn) LocalAddr() net.Addr {
	return ns.laddr
}

// RemoteAddr returns the remote network address.
func (ns *StreamConn) RemoteAddr() net.Addr {
	return ns.raddr
}
