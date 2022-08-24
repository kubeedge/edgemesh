package mdns

import (
	"bytes"
	"crypto/md5"
	"encoding/binary"
	"fmt"
	"io"
	mrand "math/rand"
	"os"

	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	maddr "github.com/multiformats/go-multiaddr"
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

// AppendMultiaddrs appending a new maddr to maddrs will consider deduplication.
func AppendMultiaddrs(maddrs *[]maddr.Multiaddr, dest maddr.Multiaddr) {
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

func AddCircuitAddrsToPeer(peer *peer.AddrInfo, relays []maddr.Multiaddr) {
	for _, relay := range relays {
		circuitAddr, err := maddr.NewMultiaddr(fmt.Sprintf("%s/p2p-circuit", relay.String()))
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
	maddrs, err := StringsToAddrs(addrs)
	if err != nil {
		return nil, err
	}
	return &peer.AddrInfo{
		ID:    pid,
		Addrs: maddrs,
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
