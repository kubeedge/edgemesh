package tunnel

import (
	"fmt"
	"net"
	"reflect"
	"testing"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
)

func TestGenerateKeyPairWithString(t *testing.T) {
	cases := []struct {
		given  string
		wanted string
	}{
		{"k8s-master", "12D3KooWB5qVCMrMNLpBDfMu6o4dy6ci2UqDVsFVomcd2PfYVzfW"},
		{"k8s-node1", "12D3KooWErZ4m27CEinjcnXNvem2KFUJUFRBcd9NdcWnYRqyr1Sn"},
		{"ke-edge1", "12D3KooWF3RB8SoMRZht7MDqF6GoipSPdFxVb9ESRLU7NuPX6ifH"},
		{"ke-edge2", "12D3KooWSD4f5fZb5c9PQ6FPVd8Em4eKX3mRezcyqXSHUyomoy8S"},
		// other form of NodeName
		{"k8s-Master", "12D3KooWA9AvBpYt17yctznPFxDvR6NFXfyu8kY72jnVLC9SKUG4"},
		{"mａsteｒ", "12D3KooWAZ7jrUXyHuTZ7rT1uUnqXZPY88HjdhvDaQjivD5M35nw"},
		{"ke-node1\n", "12D3KooWFP7gZfX1n1HLQAZt7rCZpffRvqXf7QYASxbz9mrEkAVi"},
	}
	for _, c := range cases {
		t.Run(c.given, func(t *testing.T) {
			key, err := GenerateKeyPairWithString(c.given)
			assertEqual(err, nil, t)
			out, err := peer.IDFromPrivateKey(key)
			assertEqual(err, nil, t)
			if ans := out.String(); !assertEqual(ans, c.wanted, t) {
				t.Fatalf("Node name is %s and expected key-id is %s, but %s got",
					c.given, c.wanted, ans)
			}
		})
	}

	// other situation to deal with
	cases2 := []struct {
		given  string
		wanted string
	}{
		// A null condition occurs,the answer is for null, there is no error feed back
		{"", ""},
		// illegal IP address
	}
	for _, c := range cases2 {
		t.Run(c.given, func(t *testing.T) {
			_, err := GenerateKeyPairWithString(c.given)
			assertEqual(err, fmt.Errorf("empty string"), t)
		})
	}
}

type testNode struct {
	Name string
	EIP  string
	ID   string
}

func TestGeneratePeerInfo(t *testing.T) {
	cases := []struct {
		givenNodeName string
		givenIP       string
		givenID       string
	}{
		{"k8s-master", "5.5.5.5", "12D3KooWB5qVCMrMNLpBDfMu6o4dy6ci2UqDVsFVomcd2PfYVzfW"},
		{"node1", "6.6.6.6", "12D3KooWErZ4m27CEinjcnXNvem2KFUJUFRBcd9NdcWnYRqyr1Sn"},
		{"edge1", "7.7.7.7", "12D3KooWSD4f5fZb5c9PQ6FPVd8Em4eKX3mRezcyqXSHUyomoy8S"},
		{"edge2", "8.8.8.8", "12D3KooWF3RB8SoMRZht7MDqF6GoipSPdFxVb9ESRLU7NuPX6ifH"},
	}
	for _, c := range cases {
		t.Run(c.givenNodeName, func(t *testing.T) {
			madders := make([]string, 0)
			madders = append(madders, fmt.Sprintf("/ip4/%s/tcp/9000/p2p/%s", c.givenIP, c.givenID))
			madders = append(madders, fmt.Sprintf("/ip4/%s/udp/9000/quic/p2p/%s", c.givenIP, c.givenID))
			// test if GeneratePeerInfo
			peerInfo, err := GeneratePeerInfo(c.givenNodeName, madders)
			if !assertEqual(err, nil, t) {
				t.Errorf("can not generatePeerInfo, err:%v", err)
			}
			t.Logf(peerInfo.String())
		})
	}
}

func TestAddCircuitAddrsToPeer(t *testing.T) {
	want := "{12D3KooWQ4LGgA3djuvPt4Ao9YH2U39Yge5uzn3HwHXkDK23YDVg: [/ip4/5.5.5.5/tcp/9000/p2p/12D3KooWQ4LGgA3djuvPt4Ao9YH2U39Yge5uzn3HwHXkDK23YDVg /ip4/5.5.5.5/udp/9000/quic/p2p/12D3KooWQ4LGgA3djuvPt4Ao9YH2U39Yge5uzn3HwHXkDK23YDVg /ip4/5.5.5.5/tcp/9000/p2p/12D3KooWB5qVCMrMNLpBDfMu6o4dy6ci2UqDVsFVomcd2PfYVzfW/p2p/12D3KooWQ4LGgA3djuvPt4Ao9YH2U39Yge5uzn3HwHXkDK23YDVg/p2p-circuit /ip4/5.5.5.5/udp/9000/quic/p2p/12D3KooWB5qVCMrMNLpBDfMu6o4dy6ci2UqDVsFVomcd2PfYVzfW/p2p/12D3KooWQ4LGgA3djuvPt4Ao9YH2U39Yge5uzn3HwHXkDK23YDVg/p2p-circuit /ip4/6.6.6.6/tcp/9000/p2p/12D3KooWErZ4m27CEinjcnXNvem2KFUJUFRBcd9NdcWnYRqyr1Sn/p2p/12D3KooWQ4LGgA3djuvPt4Ao9YH2U39Yge5uzn3HwHXkDK23YDVg/p2p-circuit /ip4/6.6.6.6/udp/9000/quic/p2p/12D3KooWErZ4m27CEinjcnXNvem2KFUJUFRBcd9NdcWnYRqyr1Sn/p2p/12D3KooWQ4LGgA3djuvPt4Ao9YH2U39Yge5uzn3HwHXkDK23YDVg/p2p-circuit /ip4/4.4.4.4/tcp/9000/p2p/12D3KooWSD4f5fZb5c9PQ6FPVd8Em4eKX3mRezcyqXSHUyomoy8S/p2p/12D3KooWQ4LGgA3djuvPt4Ao9YH2U39Yge5uzn3HwHXkDK23YDVg/p2p-circuit /ip4/4.4.4.4/udp/9000/quic/p2p/12D3KooWSD4f5fZb5c9PQ6FPVd8Em4eKX3mRezcyqXSHUyomoy8S/p2p/12D3KooWQ4LGgA3djuvPt4Ao9YH2U39Yge5uzn3HwHXkDK23YDVg/p2p-circuit /ip4/8.8.8.8/tcp/9000/p2p/12D3KooWF3RB8SoMRZht7MDqF6GoipSPdFxVb9ESRLU7NuPX6ifH/p2p/12D3KooWQ4LGgA3djuvPt4Ao9YH2U39Yge5uzn3HwHXkDK23YDVg/p2p-circuit /ip4/8.8.8.8/udp/9000/quic/p2p/12D3KooWF3RB8SoMRZht7MDqF6GoipSPdFxVb9ESRLU7NuPX6ifH/p2p/12D3KooWQ4LGgA3djuvPt4Ao9YH2U39Yge5uzn3HwHXkDK23YDVg/p2p-circuit /ip4/5.5.5.5/tcp/9000/p2p/12D3KooWB5qVCMrMNLpBDfMu6o4dy6ci2UqDVsFVomcd2PfYVzfW/p2p/12D3KooWQ4LGgA3djuvPt4Ao9YH2U39Yge5uzn3HwHXkDK23YDVg/p2p-circuit /ip4/5.5.5.5/udp/9000/quic/p2p/12D3KooWB5qVCMrMNLpBDfMu6o4dy6ci2UqDVsFVomcd2PfYVzfW/p2p/12D3KooWQ4LGgA3djuvPt4Ao9YH2U39Yge5uzn3HwHXkDK23YDVg/p2p-circuit /ip4/6.6.6.6/tcp/9000/p2p/12D3KooWErZ4m27CEinjcnXNvem2KFUJUFRBcd9NdcWnYRqyr1Sn/p2p/12D3KooWQ4LGgA3djuvPt4Ao9YH2U39Yge5uzn3HwHXkDK23YDVg/p2p-circuit /ip4/6.6.6.6/udp/9000/quic/p2p/12D3KooWErZ4m27CEinjcnXNvem2KFUJUFRBcd9NdcWnYRqyr1Sn/p2p/12D3KooWQ4LGgA3djuvPt4Ao9YH2U39Yge5uzn3HwHXkDK23YDVg/p2p-circuit /ip4/4.4.4.4/tcp/9000/p2p/12D3KooWSD4f5fZb5c9PQ6FPVd8Em4eKX3mRezcyqXSHUyomoy8S/p2p/12D3KooWQ4LGgA3djuvPt4Ao9YH2U39Yge5uzn3HwHXkDK23YDVg/p2p-circuit /ip4/4.4.4.4/udp/9000/quic/p2p/12D3KooWSD4f5fZb5c9PQ6FPVd8Em4eKX3mRezcyqXSHUyomoy8S/p2p/12D3KooWQ4LGgA3djuvPt4Ao9YH2U39Yge5uzn3HwHXkDK23YDVg/p2p-circuit /ip4/8.8.8.8/tcp/9000/p2p/12D3KooWF3RB8SoMRZht7MDqF6GoipSPdFxVb9ESRLU7NuPX6ifH/p2p/12D3KooWQ4LGgA3djuvPt4Ao9YH2U39Yge5uzn3HwHXkDK23YDVg/p2p-circuit /ip4/8.8.8.8/udp/9000/quic/p2p/12D3KooWF3RB8SoMRZht7MDqF6GoipSPdFxVb9ESRLU7NuPX6ifH/p2p/12D3KooWQ4LGgA3djuvPt4Ao9YH2U39Yge5uzn3HwHXkDK23YDVg/p2p-circuit]}"
	// node list
	var nodes = []*testNode{
		{Name: "k8s-master", EIP: "5.5.5.5", ID: "12D3KooWB5qVCMrMNLpBDfMu6o4dy6ci2UqDVsFVomcd2PfYVzfW"},
		{Name: "node1", EIP: "6.6.6.6", ID: "12D3KooWErZ4m27CEinjcnXNvem2KFUJUFRBcd9NdcWnYRqyr1Sn"},
		{Name: "ke-edge1", EIP: "4.4.4.4", ID: "12D3KooWSD4f5fZb5c9PQ6FPVd8Em4eKX3mRezcyqXSHUyomoy8S"},
		{Name: "ke-edge2", EIP: "8.8.8.8", ID: "12D3KooWF3RB8SoMRZht7MDqF6GoipSPdFxVb9ESRLU7NuPX6ifH"},
	}
	target := generateMulti(nodes)

	// generate relayNode
	priv, err := GenerateKeyPairWithString("MyNode")
	assertEqual(err, nil, t)
	pid, err := peer.IDFromPrivateKey(priv)
	assertEqual(err, nil, t)
	var myNodes = []*testNode{
		{Name: "MyRelayNode", EIP: "5.5.5.5", ID: pid.String()},
	}
	relay := generateMulti(myNodes)
	relayPeer := peer.AddrInfo{
		ID:    pid,
		Addrs: *relay,
	}

	relayList := RelayMap{
		"A": {ID: pid, Addrs: *target},
		"B": {ID: pid, Addrs: *target},
	}
	// test AddCircuitAddrsToPeer
	err = AddCircuitAddrsToPeer(&relayPeer, relayList)
	if err != nil {
		t.Errorf("AddAddress failed, err:%v", err)
	}
	t.Logf("the final got : %s", relayPeer.String())
	assertEqual(relayPeer.String(), want, t)
}

func TestAppendMultiaddrs(t *testing.T) {
	// generate multiAddress
	var nodes = []*testNode{
		{Name: "k8s-master", EIP: "8.8.8.8", ID: "12D3KooWB5qVCMrMNLpBDfMu6o4dy6ci2UqDVsFVomcd2PfYVzfW"},
		{Name: "ke-edge1", EIP: "6.6.6.6", ID: "12D3KooWSD4f5fZb5c9PQ6FPVd8Em4eKX3mRezcyqXSHUyomoy8S"},
	}
	node := generateMulti(nodes)
	// generate destNode
	var destNode = []*testNode{
		{"ke-edge2", "4.4.4.4", "12D3KooWSD4f5fZb5c9PQ6FPVd8Em4eKX3mRezcyqXSHUyomrfDe"},
		{"ke-edge1", "6.6.6.6", "12D3KooWSD4f5fZb5c9PQ6FPVd8Em4eKX3mRezcyqXSHUyomoy8S"},
	}
	destnode := generateMulti(destNode)

	// test if not get destination
	for _, n := range *destnode {
		*node = AppendMultiaddrs(*node, n)
	}
	Peer := peer.AddrInfo{
		ID:    "k8s",
		Addrs: *node,
	}
	t.Logf(Peer.String())
}

func assertEqual(actual, expect interface{}, t *testing.T) bool {
	if !reflect.DeepEqual(actual, expect) {
		t.Errorf("want %v, but got %v", expect, actual)
		return false
	}
	return true
}

func generateMulti(n []*testNode) *[]multiaddr.Multiaddr {
	madders := make([]string, 0)
	for _, node := range n {
		madders = append(madders, fmt.Sprintf("/ip4/%s/tcp/9000/p2p/%s", node.EIP, node.ID))
		madders = append(madders, fmt.Sprintf("/ip4/%s/udp/9000/quic/p2p/%s", node.EIP, node.ID))
	}
	maddress, err := StringsToMaddrs(madders)
	if err != nil {
		return nil
	}
	return &maddress
}

func BenchmarkGenerateKeyPairWithString(b *testing.B) {
	given := "k8s-master"
	for i := 0; i < b.N; i++ {
		_, err := GenerateKeyPairWithString(given)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGeneratePeerInfo(b *testing.B) {
	cases := []struct {
		givenNodeName string
		givenIP       string
		givenID       string
	}{
		{"k8s-master", "5.5.5.5", "12D3KooWB5qVCMrMNLpBDfMu6o4dy6ci2UqDVsFVomcd2PfYVzfW"},
		{"node1", "6.6.6.6", "12D3KooWErZ4m27CEinjcnXNvem2KFUJUFRBcd9NdcWnYRqyr1Sn"},
	}
	for _, c := range cases {
		b.Run(c.givenNodeName, func(b *testing.B) {
			madders := make([]string, 0)
			madders = append(madders, fmt.Sprintf("/ip4/%s/tcp/9000/p2p/%s", c.givenIP, c.givenID))
			madders = append(madders, fmt.Sprintf("/ip4/%s/udp/9000/quic/p2p/%s", c.givenIP, c.givenID))
			// test if GeneratePeerInfo
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := GeneratePeerInfo(c.givenNodeName, madders)
				if err != nil {
					b.Fatal(err)
				}
			}
			b.StopTimer()
		})
	}
}

func BenchmarkAppendMultiaddrs(b *testing.B) {
	// generate multiAddress
	var nodes = []*testNode{
		{Name: "k8s-master", EIP: "8.8.8.8", ID: "12D3KooWB5qVCMrMNLpBDfMu6o4dy6ci2UqDVsFVomcd2PfYVzfW"},
		{Name: "ke-edge1", EIP: "6.6.6.6", ID: "12D3KooWSD4f5fZb5c9PQ6FPVd8Em4eKX3mRezcyqXSHUyomoy8S"},
	}
	node := generateMulti(nodes)
	// generate destNode
	var destNode = []*testNode{
		{Name: "ke-edge2", EIP: "4.4.4.4", ID: "12D3KooWSD4f5fZb5c9PQ6FPVd8Em4eKX3mRezcyqXSHUyomrfDe"},
		{Name: "ke-edge1", EIP: "6.6.6.6", ID: "12D3KooWSD4f5fZb5c9PQ6FPVd8Em4eKX3mRezcyqXSHUyomoy8S"},
	}
	destnode := generateMulti(destNode)

	// test if not get destination
	b.ResetTimer()
	for _, n := range *destnode {
		for i := 0; i < b.N; i++ {
			_ = AppendMultiaddrs(*node, n)
		}
	}
	b.StopTimer()
}

func BenchmarkAddCircuitAddrsToPeer(b *testing.B) {
	priv, err := GenerateKeyPairWithString("A")
	if err != nil {
		b.Fatal(priv)
	}
	relayA, err := peer.IDFromPrivateKey(priv)
	if err != nil {
		b.Fatal(priv)
	}
	relayList := RelayMap{
		"A": {ID: relayA, Addrs: []multiaddr.Multiaddr{multiaddr.StringCast("/ip4/1.1.1.1/tcp/9000")}},
	}

	// generate relayNode
	priv, err = GenerateKeyPairWithString("MyNode")
	if err != nil {
		b.Fatal(priv)
	}
	pid, err := peer.IDFromPrivateKey(priv)
	if err != nil {
		b.Fatal(priv)
	}
	var myNodes = []*testNode{
		{Name: "MyRelayNode", EIP: "5.5.5.5", ID: pid.String()},
	}
	addrs := generateMulti(myNodes)
	pi := peer.AddrInfo{
		ID:    pid,
		Addrs: *addrs,
	}

	// test AddCircuitAddrsToPeer
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err = AddCircuitAddrsToPeer(&pi, relayList)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func TestGetIPsFromInterfaces(t *testing.T) {
	eth0, err := net.InterfaceByName("eth0")
	if err != nil {
		t.Errorf("Failed to get eth0 interface, err: %v", err)
	}
	eth0Addrs, err := eth0.Addrs()
	if err != nil {
		t.Errorf("Failed to get eth0 addrs, err: %v", err)
	}
	eth0IPs := make([]string, 0)
	for _, addr := range eth0Addrs {
		if ip, ipnet, err := net.ParseCIDR(addr.String()); err == nil && len(ipnet.Mask) == 4 {
			eth0IPs = append(eth0IPs, ip.String())
		}
	}

	tests := []struct {
		name                    string
		listenInterfaces        string
		extraFilteredInterfaces string
		want                    []string
		wantErr                 bool
	}{
		{
			name:                    "lo",
			listenInterfaces:        "lo",
			extraFilteredInterfaces: "",
			want:                    []string{"127.0.0.1"},
		},
		{
			name:                    "eth0",
			listenInterfaces:        "eth0",
			extraFilteredInterfaces: "",
			want:                    eth0IPs,
		},
		{
			name:                    "eth0 and lo",
			listenInterfaces:        "eth0,lo",
			extraFilteredInterfaces: "",
			want:                    append(eth0IPs, "127.0.0.1"),
		},
		{
			name:                    "lo and eth0",
			listenInterfaces:        "lo,eth0",
			extraFilteredInterfaces: "",
			want:                    append([]string{"127.0.0.1"}, eth0IPs...),
		},
		{
			name:                    "invalid interface name",
			listenInterfaces:        "123",
			extraFilteredInterfaces: "",
			wantErr:                 true,
		},
		{
			name:                    "eth0 and lo, lo filtered",
			listenInterfaces:        "eth0,lo",
			extraFilteredInterfaces: "lo",
			want:                    eth0IPs,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetIPsFromInterfaces(tt.listenInterfaces, tt.extraFilteredInterfaces)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetIPsFromInterfaces() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetIPsFromInterfaces() got = %v, want %v", got, tt.want)
			}
		})
	}
}
