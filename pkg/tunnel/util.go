package tunnel

import (
	"bytes"
	"crypto/md5"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	mrand "math/rand"
	"net"
	"strings"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	quic "github.com/libp2p/go-libp2p/p2p/transport/quic"
	"github.com/libp2p/go-libp2p/p2p/transport/tcp"
	ws "github.com/libp2p/go-libp2p/p2p/transport/websocket"
	ma "github.com/multiformats/go-multiaddr"
	"k8s.io/klog/v2"

	"github.com/kubeedge/edgemesh/pkg/apis/config/v1alpha1"
)

const (
	UDP       = "udp"
	TCP       = "tcp"
	Websocket = "ws"
	Quic      = "quic"

	ipIndex = 2
)

func GenerateKeyPairWithString(s string) (crypto.PrivKey, error) {
	if s == "" {
		return nil, fmt.Errorf("empty string")
	}
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

// AppendMultiaddrs append a maddr into maddrs, do nothing if contains
func AppendMultiaddrs(maddrs []ma.Multiaddr, dest ma.Multiaddr) []ma.Multiaddr {
	if !ma.Contains(maddrs, dest) {
		maddrs = append(maddrs, dest)
	}
	return maddrs
}

func AddCircuitAddrsToPeer(peer *peer.AddrInfo, relayPeers RelayMap) error {
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
		opt = libp2p.Transport(tcp.NewTCPTransport)
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

func GenerateRelayMap(relayNodes []*v1alpha1.RelayNode, protocol string, listenPort int) RelayMap {
	relayPeers := make(RelayMap)
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
	}
	return relayPeers
}

func FilterPrivateMaddr(maddrs []ma.Multiaddr) []ma.Multiaddr {
	result := make([]ma.Multiaddr, 0)
	for _, maddr := range maddrs {
		maddrElements := strings.Split(maddr.String(), "/")
		ip := maddrElements[ipIndex]
		ipAddress := net.ParseIP(ip)
		if !ipAddress.IsLoopback() && !ipAddress.IsPrivate() {
			result = append(result, maddr)
		}
	}
	return result
}

func FilterCircuitMaddr(maddrs []ma.Multiaddr) []ma.Multiaddr {
	result := make([]ma.Multiaddr, 0)
	for _, maddr := range maddrs {
		if !strings.Contains(maddr.String(), "p2p-circuit") {
			result = append(result, maddr)
		}
	}
	return result
}

func IsNoFindPeerError(err error) bool {
	return strings.HasSuffix(err.Error(), "failed to find any peer in table")
}

func GeneratePSKReader(path string) (io.Reader, error) {
	// write header
	buf := &bytes.Buffer{}
	buf.WriteString("/key/swarm/psk/1.0.0/") // pathPSKv1
	buf.WriteString("\n")
	buf.WriteString("/base64/")
	buf.WriteString("\n")

	// write encryption data
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	m := sha256.New()
	_, err = m.Write(data)
	if err != nil {
		return nil, err
	}
	key := hex.EncodeToString(m.Sum(nil))
	_, err = buf.Write([]byte(key))
	if err != nil {
		return nil, err
	}
	return buf, nil
}

// Listening to them is meaningless
var defaultFilteredInterfaces = []string{"docker", "edgemesh", "tunl", "flannel", "cni", "br", "kube-ipvs"}

func GetIPsFromInterfaces(listenInterfaces, extraFilteredInterfaces string) (ips []string, err error) {
	ifis := make([]net.Interface, 0)
	if listenInterfaces == "" || listenInterfaces == "*" {
		ifis, err = net.Interfaces()
		if err != nil {
			return nil, fmt.Errorf("failed to get all interfaces, err: %w", err)
		}
	} else {
		ifaces := strings.Split(listenInterfaces, ",")
		for _, iface := range ifaces {
			iface = strings.TrimSpace(iface)
			if iface == "" {
				continue
			}
			ifi, err := net.InterfaceByName(iface)
			if err != nil {
				return nil, fmt.Errorf("failed to get interface %s, err: %w", iface, err)
			}
			ifis = append(ifis, *ifi)
		}
	}

	// generate full filter interface list
	filteredInterfaces := defaultFilteredInterfaces
	if extraFilteredInterfaces != "" {
		for _, filterInf := range strings.Split(extraFilteredInterfaces, ",") {
			filterInf = strings.TrimSpace(filterInf)
			if filterInf != "" {
				filteredInterfaces = append(filteredInterfaces, filterInf)
			}
		}
	}

	for _, ifi := range ifis {
		if isFilterInterfaces(ifi.Name, filteredInterfaces) {
			klog.Infof("Listening to %s is meaningless, skip it.", ifi.Name)
			continue
		}
		addrs, err := ifi.Addrs()
		if err != nil {
			return nil, fmt.Errorf("failed to get interface %s addrs, err: %w", ifi.Name, err)
		}
		for _, addr := range addrs {
			if ip, ipnet, err := net.ParseCIDR(addr.String()); err == nil && len(ipnet.Mask) == 4 {
				ips = append(ips, ip.String())
			}
		}
	}
	return ips, nil
}

func isFilterInterfaces(iface string, filteredInterfaces []string) bool {
	for _, ifi := range filteredInterfaces {
		if strings.HasPrefix(iface, ifi) {
			return true
		}
	}
	return false
}
