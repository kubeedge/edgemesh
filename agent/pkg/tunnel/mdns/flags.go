package mdns

import (
	"flag"
	"fmt"
	"strings"

	"github.com/libp2p/go-libp2p-core/peer"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	maddr "github.com/multiformats/go-multiaddr"
)

// A new type we need for writing a custom flag parser
type addrList []maddr.Multiaddr

func (al *addrList) String() string {
	strs := make([]string, len(*al))
	for i, addr := range *al {
		strs[i] = addr.String()
	}
	return strings.Join(strs, ",")
}

func (al *addrList) Set(value string) error {
	addr, err := maddr.NewMultiaddr(value)
	if err != nil {
		return err
	}
	*al = append(*al, addr)
	return nil
}

func StringsToAddrs(addrStrings []string) (maddrs []maddr.Multiaddr, err error) {
	for _, addrString := range addrStrings {
		addr, err := maddr.NewMultiaddr(addrString)
		if err != nil {
			return maddrs, err
		}
		maddrs = append(maddrs, addr)
	}
	return
}

type Config struct {
	RendezvousString string
	ProtocolID       string
	UseDefault       bool
	AdvertiseAddress string

	BootstrapPeers  addrList
	ListenAddresses addrList
}

func ParseFlags() (Config, error) {
	c := Config{}
	flag.StringVar(&c.RendezvousString, "rendezvous", "my-libp2p-learning-project",
		"Unique string to identify group of nodes. Share this with your friends to let them connect with you")
	flag.Var(&c.BootstrapPeers, "peer", "Adds a peer multiaddress to the bootstrap list")
	flag.Var(&c.ListenAddresses, "listen", "Adds a multiaddress to the listen list")
	flag.StringVar(&c.ProtocolID, "pid", "/chat/1.1.0", "Sets a protocol id for stream headers")
	flag.BoolVar(&c.UseDefault, "default", true, "Use the default bootstrap peers")
	flag.StringVar(&c.AdvertiseAddress, "address", "", "Advertise address like eip")
	flag.Parse()

	if len(c.BootstrapPeers) == 0 {
		if c.UseDefault {
			c.BootstrapPeers = dht.DefaultBootstrapPeers
		} else {
			c.BootstrapPeers = relayPeers()
		}
	}

	return c, nil
}

type Node struct {
	Name string
	EIP  string
}

func (node *Node) MultiaddrsOrDie() []maddr.Multiaddr {
	addrs := make([]string, 0)
	priv, err := GenerateKeyPairWithString(node.Name)
	if err != nil {
		panic(err)
	}
	pid, err := peer.IDFromPrivateKey(priv)
	if err != nil {
		panic(err)
	}
	addrs = append(addrs, fmt.Sprintf("/ip4/%s/tcp/9000/p2p/%s", node.EIP, pid.Pretty()))
	addrs = append(addrs, fmt.Sprintf("/ip4/%s/udp/9000/quic/p2p/%s", node.EIP, pid.Pretty()))
	maddrs, err := StringsToAddrs(addrs)
	if err != nil {
		panic(err)
	}
	return maddrs
}

func relayPeers() addrList {
	// Hardcode here
	var relayNodes = []*Node{
		&Node{"k8s-master", "119.8.111.84"},
		&Node{"ke-edge1", "182.160.13.72"},
	}

	maddrs := make([]maddr.Multiaddr, 0)
	for _, node := range relayNodes {
		maddrs = append(maddrs, node.MultiaddrsOrDie()...)
	}
	return maddrs
}
