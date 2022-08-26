package tunnel

import (
	"context"
	"log"

	"github.com/libp2p/go-libp2p-core/host"
	discovery "github.com/libp2p/go-libp2p-discovery"
	dht "github.com/libp2p/go-libp2p-kad-dht"
)

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
