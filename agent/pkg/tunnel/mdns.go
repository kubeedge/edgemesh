package tunnel

import (
	"context"
	"log"

	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	discovery "github.com/libp2p/go-libp2p-discovery"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
)

type discoveryNotifee struct {
	PeerChan chan peer.AddrInfo
}

// interface to be called when new  peer is found
func (n *discoveryNotifee) HandlePeerFound(pi peer.AddrInfo) {
	n.PeerChan <- pi
}

// Initialize the MDNS service
func initMDNS(peerhost host.Host, rendezvous string) chan peer.AddrInfo {
	// register with service so that we get notified about peer discovery
	n := &discoveryNotifee{}
	n.PeerChan = make(chan peer.AddrInfo)

	// An hour might be a long long period in practical applications. But this is fine for us
	ser := mdns.NewMdnsService(peerhost, rendezvous, n)
	if err := ser.Start(); err != nil {
		panic(err)
	}
	return n.PeerChan
}

func runRoutingDiscovery(ctx context.Context, idht *dht.IpfsDHT, rendezvous string, h2 host.Host) {
	// We use a rendezvous point "meet me here" to announce our location.
	// This is like telling your friends to meet you at the Eiffel Tower.
	log.Println("Announcing ourselves...")
	routingDiscovery := discovery.NewRoutingDiscovery(idht)
	discovery.Advertise(ctx, routingDiscovery, rendezvous)
	log.Println("Successfully announced!")

	// Now, look for others who have announced
	// This is like your friend telling you the location to meet you.
	peerChan, err := routingDiscovery.FindPeers(ctx, rendezvous)
	if err != nil {
		panic(err)
	}

	for peer := range peerChan {
		if peer.ID == h2.ID() {
			continue
		}
		log.Println("Routing Found peer:", peer)
		if err = h2.Connect(ctx, peer); err != nil {
			log.Println("Connection failed:", err)
		} else {
			log.Println("Connecting to:", peer)
		}
	}
}

func runMdnsDiscovery(ctx context.Context, rendezvous string, h2 host.Host) {
	peerChan := initMDNS(h2, rendezvous)
	for peer := range peerChan {
		if peer.ID == h2.ID() {
			continue
		}
		log.Println("Mdns found peer:", peer)
		if err := h2.Connect(ctx, peer); err != nil {
			log.Println("Connection failed:", err)
		} else {
			log.Println("Connecting to:", peer)
		}
	}
}
