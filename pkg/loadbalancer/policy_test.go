package loadbalancer

import (
	"net"
	"strings"
	"testing"

	"github.com/kubeedge/edgemesh/pkg/apis/config/v1alpha1"
)

type malformedAddr string

func (a malformedAddr) Network() string { return "test" }
func (a malformedAddr) String() string  { return string(a) }

type remoteAddrConn struct {
	net.Conn
	remoteAddr net.Addr
}

func (c remoteAddrConn) RemoteAddr() net.Addr { return c.remoteAddr }

func newSourceIPPolicy() *ConsistentHashPolicy {
	config := &v1alpha1.ConsistentHash{
		PartitionCount:    100,
		ReplicationFactor: 10,
		Load:              1.25,
	}
	endpoints := []string{
		"node-a:pod-a:10.0.0.1:8080",
		"node-b:pod-b:10.0.0.2:8080",
		"node-c:pod-c:10.0.0.3:8080",
	}
	return &ConsistentHashPolicy{
		Config:   config,
		hashRing: newHashRing(config, endpoints),
		hashKey:  HashKey{Type: UserSourceIP},
	}
}

func addressesForDifferentLegacyMembers(t *testing.T, policy *ConsistentHashPolicy, ip net.IP) (*net.TCPAddr, *net.TCPAddr) {
	t.Helper()
	var firstAddr *net.TCPAddr
	var firstMember string
	for port := 1024; port <= 65535; port++ {
		addr := &net.TCPAddr{IP: ip, Port: port}
		member := policy.hashRing.LocateKey([]byte(addr.String()))
		if member == nil {
			t.Fatal("legacy address key did not locate a ring member")
		}
		if firstAddr == nil {
			firstAddr = addr
			firstMember = member.String()
			continue
		}
		if member.String() != firstMember {
			return firstAddr, addr
		}
	}
	t.Fatal("could not find source ports assigned to different legacy members")
	return nil, nil
}

func TestConsistentHashPolicyPickUsesOnlySourceIP(t *testing.T) {
	tests := []struct {
		name string
		ip   net.IP
	}{
		{name: "IPv4", ip: net.ParseIP("192.0.2.10")},
		{name: "IPv6", ip: net.ParseIP("2001:db8::10")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			policy := newSourceIPPolicy()
			firstAddr, secondAddr := addressesForDifferentLegacyMembers(t, policy, tt.ip)

			firstEndpoint, _, err := policy.Pick(nil, firstAddr, nil, nil)
			if err != nil {
				t.Fatalf("Pick() with first source address returned error: %v", err)
			}
			secondEndpoint, _, err := policy.Pick(nil, secondAddr, nil, nil)
			if err != nil {
				t.Fatalf("Pick() with second source address returned error: %v", err)
			}
			if firstEndpoint != secondEndpoint {
				t.Fatalf("same source IP selected different endpoints: %q and %q", firstEndpoint, secondEndpoint)
			}

			remoteEndpoint, _, err := policy.Pick(nil, nil, remoteAddrConn{remoteAddr: firstAddr}, nil)
			if err != nil {
				t.Fatalf("Pick() with connection remote address returned error: %v", err)
			}
			if remoteEndpoint != firstEndpoint {
				t.Fatalf("connection remote address selected endpoint %q, want %q", remoteEndpoint, firstEndpoint)
			}
		})
	}
}

func TestConsistentHashPolicyPickRejectsInvalidSourceAddress(t *testing.T) {
	tests := []struct {
		name    string
		addr    net.Addr
		conn    net.Conn
		wantErr string
	}{
		{name: "missing", wantErr: "source address is required"},
		{name: "missing remote address", conn: remoteAddrConn{}, wantErr: "source address is required"},
		{name: "malformed", addr: malformedAddr("not-a-host-port"), wantErr: "malformed source address"},
		{name: "missing host", addr: malformedAddr(":1234"), wantErr: "malformed source address"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if recovered := recover(); recovered != nil {
					t.Fatalf("Pick() panicked: %v", recovered)
				}
			}()

			policy := newSourceIPPolicy()
			_, _, err := policy.Pick(nil, tt.addr, tt.conn, nil)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Pick() error = %v, want error containing %q", err, tt.wantErr)
			}
		})
	}
}
