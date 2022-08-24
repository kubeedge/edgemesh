package tunnel

import (
	"bytes"
	"crypto/md5"
	"encoding/binary"
	"fmt"
	"io"
	mrand "math/rand"
	"os"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	quic "github.com/libp2p/go-libp2p-quic-transport"
	"github.com/libp2p/go-tcp-transport"
	ws "github.com/libp2p/go-ws-transport"
	ma "github.com/multiformats/go-multiaddr"
)

const (
	// These are the protocols supported by libp2p Transport
	TCP       = "tcp"
	Websocket = "ws"
	Quic      = "quic"
)

func HostnameOrDie() string {
	hostname, err := os.Hostname()
	if err != nil {
		panic(err)
	}
	return hostname
}

func GenerateKeyPairWithString(s string) (crypto.PrivKey, error) {
	m := md5.New()
	_, err := io.WriteString(m, s)
	if err != nil {
		return nil, err
	}
	reader := bytes.NewReader(m.Sum(nil))

	var seed int64
	err = binary.Read(reader, binary.BigEndian, &seed)
	if err != nil {
		return nil, err
	}
	r := mrand.New(mrand.NewSource(seed))
	privKey, _, err := crypto.GenerateKeyPairWithReader(crypto.Ed25519, -1, r)
	if err != nil {
		return privKey, err
	}
	return privKey, nil
}

// AppendMultiaddrs appending a new maddr to maddrs, this function will consider deduplication.
func AppendMultiaddrs(maddrs *[]ma.Multiaddr, dest ma.Multiaddr) {
	existed := false
	for _, addr := range *maddrs {
		if dest.Equal(addr) {
			existed = true
			break
		}
	}
	if !existed {
		*maddrs = append(*maddrs, dest)
	}
}

func AddCircuitAddrsToPeer(peer *peer.AddrInfo, relays []ma.Multiaddr) {
	for _, relay := range relays {
		circuitAddr, err := ma.NewMultiaddr(fmt.Sprintf("%s/p2p-circuit", relay.String()))
		if err != nil {
			panic(err)
		}
		peer.Addrs = append(peer.Addrs, circuitAddr)
	}
}

func GeneratePeerInfo(hostname string, addrs []string) (*peer.AddrInfo, error) {
	priv, err := GenerateKeyPairWithString(hostname)
	if err != nil {
		return nil, err
	}
	pid, err := peer.IDFromPrivateKey(priv)
	if err != nil {
		return nil, err
	}
	mas, err := StringsToMaddrs(addrs)
	if err != nil {
		return nil, err
	}
	return &peer.AddrInfo{
		ID:    pid,
		Addrs: mas,
	}, nil

}

func PeerIDFromString(s string) (peer.ID, error) {
	privKey, err := GenerateKeyPairWithString(s)
	if err != nil {
		return "", err
	}
	peerid, err := peer.IDFromPrivateKey(privKey)
	if err != nil {
		return "", err
	}
	return peerid, nil
}

// GenerateTransportOption generate Transport option from protocol
// supported protocols: ["tcp", "ws", "quic"]
func GenerateTransportOption(protocol string) libp2p.Option {
	var opt libp2p.Option
	switch protocol {
	case TCP:
		libp2p.Transport(tcp.NewTCPTransport)
	case Websocket:
		opt = libp2p.Transport(ws.New)
	case Quic:
		opt = libp2p.Transport(quic.NewTransport)
	}
	return opt
}

// GenerateMultiAddrString generate an IPv4 multi-address string by protocol, ip and port
// supported protocols: ["tcp", "ws", "quic"]
func GenerateMultiAddrString(protocol, ip string, port int) string {
	var maddr string
	switch protocol {
	case TCP:
		maddr = fmt.Sprintf("/ip4/%s/tcp/%d", ip, port)
	case Websocket:
		maddr = fmt.Sprintf("/ip4/%s/tcp/%d/ws", ip, port)
	case Quic:
		maddr = fmt.Sprintf("/ip4/%s/udp/%d/quic", ip, port)
	}
	return maddr
}

// StringsToMaddrs convert multi-address strings to Maddrs
func StringsToMaddrs(addrStrings []string) (mas []ma.Multiaddr, err error) {
	for _, addrString := range addrStrings {
		addr, err := ma.NewMultiaddr(addrString)
		if err != nil {
			return mas, err
		}
		mas = append(mas, addr)
	}
	return
}
