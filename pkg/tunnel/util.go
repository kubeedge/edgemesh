package tunnel

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/binary"
	"fmt"
	"io"
	mrand "math/rand"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kubeedge/edgemesh/pkg/apis/config/v1alpha1"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/crypto"
	p2phost "github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	quic "github.com/libp2p/go-libp2p-quic-transport"
	"github.com/libp2p/go-tcp-transport"
	ws "github.com/libp2p/go-ws-transport"
	ma "github.com/multiformats/go-multiaddr"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
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

func AddCircuitAddrsToPeer(peer *peer.AddrInfo, relayPeers map[string]*peer.AddrInfo) error {
	for _, relay := range relayPeers {
		for _, maddr := range relay.Addrs {
			circuitAddr, err := ma.NewMultiaddr(strings.Join([]string{maddr.String(), "p2p", relay.ID.String(), "p2p-circuit"}, "/"))
			if err != nil {
				return err
			}
			peer.Addrs = append(peer.Addrs, circuitAddr)
		}
	}
	return nil
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

func GenerateRelayRecord(relayNodes []*v1alpha1.RelayNode, protocol string, listenPort int) (map[string]*peer.AddrInfo, []ma.Multiaddr) {
	relayPeers := make(map[string]*peer.AddrInfo)
	relayMaddrs := make([]ma.Multiaddr, 0)
	for _, relayNode := range relayNodes {
		nodeName := relayNode.NodeName
		peerid, err := PeerIDFromString(nodeName)
		if err != nil {
			klog.Errorf("Failed to generate peer id from %s", nodeName)
			continue
		}
		// TODO It is assumed here that we have checked the validity of the IP.
		addrStrings := make([]string, 0)
		for _, addr := range relayNode.AdvertiseAddress {
			addrStrings = append(addrStrings, GenerateMultiAddrString(protocol, addr, listenPort))
		}
		maddrs, err := StringsToMaddrs(addrStrings)
		if err != nil {
			klog.Errorf("Failed to convert addr strings to maddrs: %v", err)
			continue
		}
		relayPeers[nodeName] = &peer.AddrInfo{
			ID:    peerid,
			Addrs: maddrs,
		}
		relayMaddrs = append(relayMaddrs, maddrs...)
	}
	return relayPeers, relayMaddrs
}

func BootstrapConnect(ctx context.Context, ph p2phost.Host, peers map[string]*peer.AddrInfo) error {
	return wait.PollImmediate(5*time.Second, 2*time.Minute, func() (bool, error) { // todo get timeout from config
		var count int32
		var wg sync.WaitGroup
		for _, p := range peers {
			if p.ID == ph.ID() {
				atomic.AddInt32(&count, 1)
				continue
			}

			wg.Add(1)
			go func(p *peer.AddrInfo) {
				defer wg.Done()
				defer klog.Infoln("bootstrapDial", ph.ID(), p.ID)
				klog.Infof("%s bootstrapping to %s", ph.ID(), p.ID)

				ph.Peerstore().AddAddrs(p.ID, p.Addrs, peerstore.PermanentAddrTTL)
				if err := ph.Connect(ctx, *p); err != nil {
					klog.Infoln("bootstrapDialFailed", p.ID)
					klog.Infof("failed to bootstrap with %v: %s", p.ID, err)
					return
				}
				klog.Infoln("bootstrapDialSuccess", p.ID)
				klog.Infof("bootstrapped with %v", p.ID)
				atomic.AddInt32(&count, 1)
			}(p)
		}
		wg.Wait()
		if count != int32(len(peers)) {
			klog.Errorf("Not all bootstrapDail connected, continue bootstrapDail...")
			return false, nil
		}
		return true, nil
	})
}
