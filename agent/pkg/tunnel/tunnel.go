package tunnel

import (
	"context"
	"encoding/json"
	"time"

	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
	"k8s.io/klog/v2"

	"github.com/kubeedge/edgemesh/agent/pkg/tunnel/controller"
	"github.com/kubeedge/edgemesh/common/constants"
)

const RETRY_CONNECT_TIME = 3

var peerAddrInfo = &peer.AddrInfo{}

func (t *TunnelAgent) Run() {
	isConnected := false
	p2pProto := ma.ProtocolWithCode(ma.P_P2P)
	circuitProto := ma.ProtocolWithCode(ma.P_CIRCUIT)
	for {
		relays, err := controller.APIConn.GetPeerAddrInfo(constants.ServerAddrName)
		if err != nil {
			klog.Errorln("Failed to get tunnel server addr")
			time.Sleep(5 * time.Second)
			continue
		}
		for _, res := range relays {
			data, ok := res.(map[string]interface{})
			if !ok {
				klog.Errorf("Failed to get addr, err: %v", res)
				continue
			}
			relay := new(peer.AddrInfo)
			dataType, err := json.Marshal(data)
			if err != nil {
				klog.Warningf("Marshal addr err: %v", err)
				continue
			}
			err = relay.UnmarshalJSON(dataType)
			if err != nil {
				klog.Errorf("UnmarshalJSON addr err: %v", err)
				continue
			}
			for _, v := range relay.Addrs {
				circuitAddr, err := ma.NewMultiaddr(v.String() + "/" + p2pProto.Name + "/" + relay.ID.String() + "/" + circuitProto.Name)
				if err != nil {
					klog.Warningf("New multi addr err: %v", err)
					continue
				}
				peerAddrInfo.Addrs = append(peerAddrInfo.Addrs, circuitAddr)
			}
			if !isConnected {
				isConnected = processConnection(t.Host, relay)
			}
		}
		if !isConnected {
			continue
		}
		for {
			if t.Mode == ServerMode || t.Mode == ServerClientMode {
				err := controller.APIConn.SetPeerAddrInfo(t.Config.NodeName, peerAddrInfo)
				if err != nil {
					klog.Warningf("Set peer addr info to secret err: %v", err)
					time.Sleep(10 * time.Second)
					continue
				}
				break
			}
		}
		// heartbeat time
		time.Sleep(10 * time.Second)
	}
}

func processConnection(host host.Host, relay *peer.AddrInfo) bool {
	if len(host.Network().ConnsToPeer(relay.ID)) == 0 {
		klog.Warningf("Connection between agent and server %v is not established, try connect", relay.Addrs)
		retryTime := 0
		for retryTime < RETRY_CONNECT_TIME {
			klog.Infof("Tunnel agent connecting to tunnel server")
			err := host.Connect(context.Background(), *relay)
			if err != nil {
				klog.Warningf("Connect to server err: %v", err)
				time.Sleep(2 * time.Second)
				retryTime++
				continue
			}
			peerAddrInfo.ID = host.ID()
			peerAddrInfo.Addrs = append(peerAddrInfo.Addrs, host.Addrs()...)
			klog.Infof("agent success connected to server %v", relay.Addrs)
			return true
		}
	}
	return false
}
