package tunnel

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
)

/**
GenerateKeyPairWithString should generate Key with Nodename
*/
func TestGenerateKeyPairWithString(t *testing.T) {
	//Benchmarks to identify different node names, including numbers and letters
	cases := []struct {
		given, wanted string
	}{
		{"k8s-master", "12D3KooWB5qVCMrMNLpBDfMu6o4dy6ci2UqDVsFVomcd2PfYVzfW"},
		{"k8s-node1", "12D3KooWErZ4m27CEinjcnXNvem2KFUJUFRBcd9NdcWnYRqyr1Sn"},
		{"ke-edge1", "12D3KooWF3RB8SoMRZht7MDqF6GoipSPdFxVb9ESRLU7NuPX6ifH"},
		{"ke-edge2", "12D3KooWSD4f5fZb5c9PQ6FPVd8Em4eKX3mRezcyqXSHUyomoy8S"},
		//other form of NodeName
		{"k8s-Master", "12D3KooWB5qVCMrMNLpBDfMu6o4dy6ci2UqDVsFVomcd2PfYVzfW"},
		{"mａsteｒ", "12D3KooWB5qVCMrMNLpBDfMu6o4dy6ci2UqDVsFVomcd2PfYVzfW"},
		{"ke-node1\n", "12D3KooWErZ4m27CEinjcnXNvem2KFUJUFRBcd9NdcWnYRqyr1Sn"},
	}
	for _, c := range cases {
		t.Run(c.given, func(t *testing.T) {
			key, err := GenerateKeyPairWithString(c.given)
			assertEqual(err, nil, t)
			out, err := peer.IDFromPrivateKey(key)
			assertEqual(err, nil, t)
			if ans := peer.Encode(out); !assertEqual(ans, c.wanted, t) {
				t.Fatalf("Node name is %s and expected key-id is %s, but %s got",
					c.given, c.wanted, ans)
			}
		})
	}
	// other situation to deal with

	cases2 := []struct {
		given, wanted string
	}{
		//A null condition occurs,the answer is for null ,there is no err fead back
		{"", ""},
		//illegal IP address
	}
	for _, c := range cases2 {
		t.Run(c.given, func(t *testing.T) {
			key, err := GenerateKeyPairWithString(c.given)
			if assertEqual(err, nil, t) {
				out, err := peer.IDFromPrivateKey(key)
				assertEqual(err, nil, t)
				t.Fatalf("Node name is nil but still got Key-ID: %s",
					peer.Encode(out))
			}
		})
	}

}

/**
AddCircuitAddrsToPeer Get a peerInfo and the list of relay server-Address
and manually the '/p2p-circuit' into the address
*/
type testNode struct {
	Name string
	EIP  string
	ID   string
}

func TestGeneratePeerInfo(t *testing.T) {
	cases := []struct {
		givenNodename, givenIP, givenID string
	}{
		{"k8s-master", "5.5.5.5", "12D3KooWB5qVCMrMNLpBDfMu6o4dy6ci2UqDVsFVomcd2PfYVzfW"},
		{"node1", "6.6.6.6", "12D3KooWErZ4m27CEinjcnXNvem2KFUJUFRBcd9NdcWnYRqyr1Sn"},
		{"edge1", "7.7.7.7", "12D3KooWSD4f5fZb5c9PQ6FPVd8Em4eKX3mRezcyqXSHUyomoy8S"},
		{"edge2", "8.8.8.8", "12D3KooWF3RB8SoMRZht7MDqF6GoipSPdFxVb9ESRLU7NuPX6ifH"},
		// other illegal to detect
		{"k8s-master", "888.888.888.888", "12D3KooWB5qVCMrMNLpBDfMu6o4dy6ci2UqDVsFVomcd2PfYVzfW"},
		{"k8s-master", "www.k8s-master.com", "12D3KooWB5qVCMrMNLpBDfMu6o4dy6ci2UqDVsFVomcd2PfYVzfW"},
	}
	for _, c := range cases {
		t.Run(c.givenNodename, func(t *testing.T) {
			madders := make([]string, 0)
			madders = append(madders, fmt.Sprintf("/ip4/%s/tcp/9000/p2p/%s", c.givenIP, c.givenID))
			madders = append(madders, fmt.Sprintf("/ip4/%s/udp/9000/quic/p2p/%s", c.givenIP, c.givenID))
			//test if GeneratePeerInfo
			peerInfo, err := GeneratePeerInfo(c.givenNodename, madders)
			if !assertEqual(err, nil, t) {
				t.Errorf("can not generatePeerInfo, err:%v", err)
			}
			t.Logf(peerInfo.String())
		})
	}
}

func TestAddCircuitAddrsToPeer(t *testing.T) {
	want := "{12D3KooWQ4LGgA3djuvPt4Ao9YH2U39Yge5uzn3HwHXkDK23YDVg: [/ip4/5.5.5.5/tcp/9000/p2p/12D3KooWQ4LGgA3djuvPt4Ao9YH2U39Yge5uzn3HwHXkDK23YDVg /ip4/5.5.5.5/udp/9000/quic/p2p/12D3KooWQ4LGgA3djuvPt4Ao9YH2U39Yge5uzn3HwHXkDK23YDVg /ip4/5.5.5.5/tcp/9000/p2p/12D3KooWB5qVCMrMNLpBDfMu6o4dy6ci2UqDVsFVomcd2PfYVzfW/p2p/12D3KooWQ4LGgA3djuvPt4Ao9YH2U39Yge5uzn3HwHXkDK23YDVg/p2p-circuit /ip4/5.5.5.5/udp/9000/quic/p2p/12D3KooWB5qVCMrMNLpBDfMu6o4dy6ci2UqDVsFVomcd2PfYVzfW/p2p/12D3KooWQ4LGgA3djuvPt4Ao9YH2U39Yge5uzn3HwHXkDK23YDVg/p2p-circuit /ip4/6.6.6.6/tcp/9000/p2p/12D3KooWErZ4m27CEinjcnXNvem2KFUJUFRBcd9NdcWnYRqyr1Sn/p2p/12D3KooWQ4LGgA3djuvPt4Ao9YH2U39Yge5uzn3HwHXkDK23YDVg/p2p-circuit /ip4/6.6.6.6/udp/9000/quic/p2p/12D3KooWErZ4m27CEinjcnXNvem2KFUJUFRBcd9NdcWnYRqyr1Sn/p2p/12D3KooWQ4LGgA3djuvPt4Ao9YH2U39Yge5uzn3HwHXkDK23YDVg/p2p-circuit /ip4/4.4.4.4/tcp/9000/p2p/12D3KooWSD4f5fZb5c9PQ6FPVd8Em4eKX3mRezcyqXSHUyomoy8S/p2p/12D3KooWQ4LGgA3djuvPt4Ao9YH2U39Yge5uzn3HwHXkDK23YDVg/p2p-circuit /ip4/4.4.4.4/udp/9000/quic/p2p/12D3KooWSD4f5fZb5c9PQ6FPVd8Em4eKX3mRezcyqXSHUyomoy8S/p2p/12D3KooWQ4LGgA3djuvPt4Ao9YH2U39Yge5uzn3HwHXkDK23YDVg/p2p-circuit /ip4/8.8.8.8/tcp/9000/p2p/12D3KooWF3RB8SoMRZht7MDqF6GoipSPdFxVb9ESRLU7NuPX6ifH/p2p/12D3KooWQ4LGgA3djuvPt4Ao9YH2U39Yge5uzn3HwHXkDK23YDVg/p2p-circuit /ip4/8.8.8.8/udp/9000/quic/p2p/12D3KooWF3RB8SoMRZht7MDqF6GoipSPdFxVb9ESRLU7NuPX6ifH/p2p/12D3KooWQ4LGgA3djuvPt4Ao9YH2U39Yge5uzn3HwHXkDK23YDVg/p2p-circuit /ip4/5.5.5.5/tcp/9000/p2p/12D3KooWB5qVCMrMNLpBDfMu6o4dy6ci2UqDVsFVomcd2PfYVzfW/p2p/12D3KooWQ4LGgA3djuvPt4Ao9YH2U39Yge5uzn3HwHXkDK23YDVg/p2p-circuit /ip4/5.5.5.5/udp/9000/quic/p2p/12D3KooWB5qVCMrMNLpBDfMu6o4dy6ci2UqDVsFVomcd2PfYVzfW/p2p/12D3KooWQ4LGgA3djuvPt4Ao9YH2U39Yge5uzn3HwHXkDK23YDVg/p2p-circuit /ip4/6.6.6.6/tcp/9000/p2p/12D3KooWErZ4m27CEinjcnXNvem2KFUJUFRBcd9NdcWnYRqyr1Sn/p2p/12D3KooWQ4LGgA3djuvPt4Ao9YH2U39Yge5uzn3HwHXkDK23YDVg/p2p-circuit /ip4/6.6.6.6/udp/9000/quic/p2p/12D3KooWErZ4m27CEinjcnXNvem2KFUJUFRBcd9NdcWnYRqyr1Sn/p2p/12D3KooWQ4LGgA3djuvPt4Ao9YH2U39Yge5uzn3HwHXkDK23YDVg/p2p-circuit /ip4/4.4.4.4/tcp/9000/p2p/12D3KooWSD4f5fZb5c9PQ6FPVd8Em4eKX3mRezcyqXSHUyomoy8S/p2p/12D3KooWQ4LGgA3djuvPt4Ao9YH2U39Yge5uzn3HwHXkDK23YDVg/p2p-circuit /ip4/4.4.4.4/udp/9000/quic/p2p/12D3KooWSD4f5fZb5c9PQ6FPVd8Em4eKX3mRezcyqXSHUyomoy8S/p2p/12D3KooWQ4LGgA3djuvPt4Ao9YH2U39Yge5uzn3HwHXkDK23YDVg/p2p-circuit /ip4/8.8.8.8/tcp/9000/p2p/12D3KooWF3RB8SoMRZht7MDqF6GoipSPdFxVb9ESRLU7NuPX6ifH/p2p/12D3KooWQ4LGgA3djuvPt4Ao9YH2U39Yge5uzn3HwHXkDK23YDVg/p2p-circuit /ip4/8.8.8.8/udp/9000/quic/p2p/12D3KooWF3RB8SoMRZht7MDqF6GoipSPdFxVb9ESRLU7NuPX6ifH/p2p/12D3KooWQ4LGgA3djuvPt4Ao9YH2U39Yge5uzn3HwHXkDK23YDVg/p2p-circuit]}"
	//节点列表
	var Nodes = []*testNode{
		&testNode{"k8s-master", "5.5.5.5", "12D3KooWB5qVCMrMNLpBDfMu6o4dy6ci2UqDVsFVomcd2PfYVzfW"},
		&testNode{"node1", "6.6.6.6", "12D3KooWErZ4m27CEinjcnXNvem2KFUJUFRBcd9NdcWnYRqyr1Sn"},
		&testNode{"ke-edge1", "4.4.4.4", "12D3KooWSD4f5fZb5c9PQ6FPVd8Em4eKX3mRezcyqXSHUyomoy8S"},
		&testNode{"ke-edge2", "8.8.8.8", "12D3KooWF3RB8SoMRZht7MDqF6GoipSPdFxVb9ESRLU7NuPX6ifH"},
	}
	target := generatemulti(Nodes, t)

	// generate relayNode
	priv, err := GenerateKeyPairWithString("MyNode")
	assertEqual(err, nil, t)
	pid, err := peer.IDFromPrivateKey(priv)
	assertEqual(err, nil, t)
	var MyNodes = []*testNode{&testNode{"MyRelayNode", "5.5.5.5", pid.Pretty()}}
	relay := generatemulti(MyNodes, t)
	relayPeer := peer.AddrInfo{
		ID:    pid,
		Addrs: *relay,
	}

	relayList := RelayMap{
		"A": {pid, *target},
		"B": {pid, *target},
	}
	//test AddCircuitAddrsToPeer
	AddCircuitAddrsToPeer(&relayPeer, relayList)
	if err != nil {
		t.Errorf("AddAddress failed, err:%v", err)
	}
	t.Logf("the final got : %s", relayPeer.String())
	assertEqual(relayPeer.String(), want, t)
}

func TestAppendMultiaddrs(t *testing.T) {
	//want :=

	// generate multiAddress
	var Nodes = []*testNode{
		&testNode{"k8s-master", "8.8.8.8", "12D3KooWB5qVCMrMNLpBDfMu6o4dy6ci2UqDVsFVomcd2PfYVzfW"},
		&testNode{"ke-edge1", "6.6.6.6", "12D3KooWSD4f5fZb5c9PQ6FPVd8Em4eKX3mRezcyqXSHUyomoy8S"},
	}
	node := generatemulti(Nodes, t)
	// generate destNode
	var destNode = []*testNode{
		&testNode{"ke-edge2", "4.4.4.4", "12D3KooWSD4f5fZb5c9PQ6FPVd8Em4eKX3mRezcyqXSHUyomrfDe"},
		&testNode{"ke-edge1", "6.6.6.6", "12D3KooWSD4f5fZb5c9PQ6FPVd8Em4eKX3mRezcyqXSHUyomoy8S"},
	}
	destnode := generatemulti(destNode, t)

	//test if not get destination
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

func assertTrue(t *testing.T, a bool) {
	t.Helper()
	if !a {
		t.Errorf("Not True %t", a)
	}
}

func assertFalse(t *testing.T, a bool) {
	t.Helper()
	if a {
		t.Errorf("Not True %t", a)
	}
}

func assertNil(t *testing.T, a interface{}) {
	t.Helper()
	if a != nil {
		t.Error("Not Nil")
	}
}

func assertNotNil(t *testing.T, a interface{}) {
	t.Helper()
	if a == nil {
		t.Error("Is Nil")
	}
}

func generatemulti(n []*testNode, t *testing.T) *[]multiaddr.Multiaddr {
	madders := make([]string, 0)
	for _, node := range n {
		madders = append(madders, fmt.Sprintf("/ip4/%s/tcp/9000/p2p/%s", node.EIP, node.ID))
		madders = append(madders, fmt.Sprintf("/ip4/%s/udp/9000/quic/p2p/%s", node.EIP, node.ID))
	}
	maddress, err := StringsToMaddrs(madders)
	assertEqual(err, nil, t)
	return &maddress
}

func BenchmarkGenerateKeyPairWithString(b *testing.B) {
	given := "k8s-master"
	b.ResetTimer()
	key, err := GenerateKeyPairWithString(given)
	b.StopTimer()
	assertEqualB(err, nil, b)
	out, err := peer.IDFromPrivateKey(key)
	assertEqualB(err, nil, b)
	b.Logf("we got the key is :%s", out.String())
}

func BenchmarkGeneratePeerInfo(b *testing.B) {
	cases := []struct {
		givenNodename, givenIP, givenID string
	}{
		{"k8s-master", "5.5.5.5", "12D3KooWB5qVCMrMNLpBDfMu6o4dy6ci2UqDVsFVomcd2PfYVzfW"},
		{"node1", "6.6.6.6", "12D3KooWErZ4m27CEinjcnXNvem2KFUJUFRBcd9NdcWnYRqyr1Sn"},
	}
	for _, c := range cases {
		b.Run(c.givenNodename, func(b *testing.B) {
			madders := make([]string, 0)
			madders = append(madders, fmt.Sprintf("/ip4/%s/tcp/9000/p2p/%s", c.givenIP, c.givenID))
			madders = append(madders, fmt.Sprintf("/ip4/%s/udp/9000/quic/p2p/%s", c.givenIP, c.givenID))
			//test if GeneratePeerInfo
			b.ResetTimer()
			peerInfo, err := GeneratePeerInfo(c.givenNodename, madders)
			b.StopTimer()
			if !assertEqualB(err, nil, b) {
				b.Errorf("can not generatePeerInfo, err:%v", err)
			}
			b.Logf(peerInfo.String())
		})
	}
}

func BenchmarkAppendMultiaddrs(b *testing.B) {
	// generate multiAddress
	var Nodes = []*testNode{
		&testNode{"k8s-master", "8.8.8.8", "12D3KooWB5qVCMrMNLpBDfMu6o4dy6ci2UqDVsFVomcd2PfYVzfW"},
		&testNode{"ke-edge1", "6.6.6.6", "12D3KooWSD4f5fZb5c9PQ6FPVd8Em4eKX3mRezcyqXSHUyomoy8S"},
	}
	node := generatemultiB(Nodes, b)
	// generate destNode
	var destNode = []*testNode{
		&testNode{"ke-edge2", "4.4.4.4", "12D3KooWSD4f5fZb5c9PQ6FPVd8Em4eKX3mRezcyqXSHUyomrfDe"},
		&testNode{"ke-edge1", "6.6.6.6", "12D3KooWSD4f5fZb5c9PQ6FPVd8Em4eKX3mRezcyqXSHUyomoy8S"},
	}
	destnode := generatemultiB(destNode, b)

	//test if not get destination
	b.ResetTimer()
	for _, n := range *destnode {
		*node = AppendMultiaddrs(*node, n)
	}
	b.StopTimer()
}

func BenchmarkAddCircuitAddrsToPeer(b *testing.B) {
	//节点列表
	var Nodes = []*testNode{
		&testNode{"k8s-master", "5.5.5.5", "12D3KooWB5qVCMrMNLpBDfMu6o4dy6ci2UqDVsFVomcd2PfYVzfW"},
		&testNode{"node1", "6.6.6.6", "12D3KooWErZ4m27CEinjcnXNvem2KFUJUFRBcd9NdcWnYRqyr1Sn"},
		&testNode{"ke-edge1", "4.4.4.4", "12D3KooWSD4f5fZb5c9PQ6FPVd8Em4eKX3mRezcyqXSHUyomoy8S"},
		&testNode{"ke-edge2", "8.8.8.8", "12D3KooWF3RB8SoMRZht7MDqF6GoipSPdFxVb9ESRLU7NuPX6ifH"},
	}
	target := generatemultiB(Nodes, b)

	relayList := RelayMap{
		"A": {"12D3KooWF3RB8SoMRZht7MDqF6GoipSPdFxVb9EikoU7NuPX6hjk", *target},
		"B": {"12D3KooWF3RB8SoMRZht7MDqF3RB8SSPdFxVb9EikoU7NuPXjkoP", *target},
	}

	// generate relayNode
	priv, err := GenerateKeyPairWithString("MyNode")
	assertEqualB(err, nil, b)
	pid, err := peer.IDFromPrivateKey(priv)
	assertEqualB(err, nil, b)
	var MyNodes = []*testNode{
		&testNode{"MyRelayNode", "5.5.5.5", pid.Pretty()},
	}
	relay := generatemultiB(MyNodes, b)
	relayPeer := peer.AddrInfo{
		ID:    pid,
		Addrs: *relay,
	}

	//test AddCircuitAddrsToPeer
	b.ResetTimer()
	AddCircuitAddrsToPeer(&relayPeer, relayList)
	b.StopTimer()
	if err != nil {
		b.Errorf("AddAddress failed, err:%v", err)
	}
	b.Logf(relayPeer.String())
	//b.Logf("the final got : %s", relayPeer.String())
}

func assertEqualB(actual, expect interface{}, b *testing.B) bool {
	if !reflect.DeepEqual(actual, expect) {
		b.Errorf("want %v, but got %v", expect, actual)
		return false
	}
	return true
}

func generatemultiB(n []*testNode, b *testing.B) *[]multiaddr.Multiaddr {
	madders := make([]string, 0)
	for _, node := range n {
		madders = append(madders, fmt.Sprintf("/ip4/%s/tcp/9000/p2p/%s", node.EIP, node.ID))
		madders = append(madders, fmt.Sprintf("/ip4/%s/udp/9000/quic/p2p/%s", node.EIP, node.ID))
	}
	maddress, err := StringsToMaddrs(madders)
	assertEqualB(err, nil, b)
	return &maddress
}
