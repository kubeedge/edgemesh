package tunnel

import (
	"context"
	"time"

	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
	"k8s.io/klog/v2"

	"github.com/kubeedge/edgemesh/agent/pkg/tunnel/controller"
	"github.com/kubeedge/edgemesh/common/constants"
)

const connectRetryTime = 3
const runRetryTime = 10

func (t *TunnelAgent) Run() {
	isSuccess := false
	for i := 0; i < runRetryTime; i++ {
		relay, err := controller.APIConn.GetPeerAddrInfo(constants.ServerAddrName)
		if err != nil {
			klog.Errorf("Failed to get tunnel server addr: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}

		if len(t.Host.Network().ConnsToPeer(relay.ID)) == 0 {
			klog.Warningf("Connection between agent and server %v is not established, try connect", relay.Addrs)
			retryTime := 0
			for retryTime < connectRetryTime {
				klog.Infof("Tunnel agent connecting to tunnel server")
				err = t.Host.Connect(context.Background(), *relay)
				if err != nil {
					klog.Warningf("Connect to server err: %v", err)
					time.Sleep(2 * time.Second)
					retryTime++
					continue
				}

				if t.Mode == ServerMode || t.Mode == ServerClientMode {
					err = controller.APIConn.SetPeerAddrInfo(t.Config.NodeName, InfoFromHostAndRelay(t.Host, relay))
					if err != nil {
						klog.Warningf("Set peer addr info to secret err: %v", err)
						time.Sleep(2 * time.Second)
						retryTime++
						continue
					}
				}

				klog.Infof("agent success connected to server %v", relay.Addrs)
				isSuccess = true
				break
			}
		}
	}
	if !isSuccess {
		klog.Fatal("Failed to run tunnelAgent, exit")
	}
}

func InfoFromHostAndRelay(host host.Host, relay *peer.AddrInfo) *peer.AddrInfo {
	p2pProto := ma.ProtocolWithCode(ma.P_P2P)
	circuitProto := ma.ProtocolWithCode(ma.P_CIRCUIT)
	peerAddrInfo := &peer.AddrInfo{
		ID:    host.ID(),
		Addrs: host.Addrs(),
	}
	for _, v := range relay.Addrs {
		circuitAddr, err := ma.NewMultiaddr(v.String() + "/" + p2pProto.Name + "/" + relay.ID.String() + "/" + circuitProto.Name)
		if err != nil {
			klog.Warningf("New multi addr err: %v", err)
			continue
		}
		peerAddrInfo.Addrs = append(peerAddrInfo.Addrs, circuitAddr)
	}
	return peerAddrInfo
}
