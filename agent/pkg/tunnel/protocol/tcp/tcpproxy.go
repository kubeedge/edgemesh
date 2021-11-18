package tcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/libp2p/go-msgio/protoio"
	"k8s.io/klog/v2"

	"github.com/kubeedge/edgemesh/agent/pkg/tunnel/controller"
	tcp_pb "github.com/kubeedge/edgemesh/agent/pkg/tunnel/protocol/tcp/pb"
	"github.com/kubeedge/edgemesh/common/constants"
)

const (
	MAX_READ_SIZE           = 4 * 1024
	MAX_RETRY_CONNCECT_TIME = 3
)

var TCPProxyProtocol protocol.ID = "/libp2p/tcpproxy/1.0.0"

type TCPProxyService struct {
	host host.Host
}

func NewTCPProxyService(h host.Host) *TCPProxyService {
	return &TCPProxyService{
		host: h,
	}
}

func (tp *TCPProxyService) ProxyStreamHandler(s network.Stream) {
	// todo use peerID to get nodeName
	klog.Infof("Get a new stream from %s", s.Conn().RemotePeer().String())
	streamWriter := protoio.NewDelimitedWriter(s)
	streamReader := protoio.NewDelimitedReader(s, MAX_READ_SIZE)

	msg := new(tcp_pb.TCPProxy)
	err := streamReader.ReadMsg(msg)
	if err != nil {
		klog.Errorf("Read msg from %s err: %v", s.Conn().RemotePeer().String(), err)
		return
	}
	if msg.GetType() != tcp_pb.TCPProxy_CONNECT {
		klog.Errorf("Read msg from %s type should be CONNECT", s.Conn().RemotePeer().String())
		return
	}

	targetAddr := &net.TCPAddr{
		IP:   net.ParseIP(msg.GetIp()),
		Port: int(msg.GetPort()),
	}
	klog.Infof("l4 proxy get tcp server address: %v", targetAddr)

	var proxyClient net.Conn
	for i := 0; i < MAX_RETRY_CONNCECT_TIME; i++ {
		proxyClient, err = net.DialTimeout("tcp", targetAddr.String(), 5*time.Second)
		if err == nil {
			break
		}
	}
	if err != nil {
		klog.Errorf("l4 proxy connect to %v err: %v", targetAddr, err)
		msg.Reset()
		msg.Type = tcp_pb.TCPProxy_FAILED.Enum()
		if err := streamWriter.WriteMsg(msg); err != nil {
			klog.Errorf("Write msg to %s err: %v", s.Conn().RemotePeer().String(), err)
			return
		}
		return
	}

	msg.Reset()
	msg.Type = tcp_pb.TCPProxy_SUCCESS.Enum()
	if err = streamWriter.WriteMsg(msg); err != nil {
		klog.Errorf("Write msg to %s err: %v", s.Conn().RemotePeer().String(), err)
		return
	}

	msg.Reset()

	close := sync.Once{}
	cp := func(dst io.WriteCloser, src io.ReadCloser) {
		_, err = io.Copy(dst, src)
		if err != nil && err != io.EOF && !strings.Contains(err.Error(), constants.ConnectionClosed) && !strings.Contains(err.Error(), constants.StreamReset) {
			klog.Errorf("io copy between proxy and client error: %v", err)
		}
		close.Do(func() {
			dst.Close()
			src.Close()
		})
	}

	go cp(proxyClient, s)
	cp(s, proxyClient)

	klog.Infof("Success proxy for targetAddr: %v", targetAddr)
}

func (tp *TCPProxyService) GetProxyStream(targetNodeName, targetIP string, targetPort int32) (io.ReadWriteCloser, error) {
	destInfo, err := controller.APIConn.GetPeerAddrInfo(targetNodeName)
	if err != nil {
		return nil, fmt.Errorf("Get %s addr err: %v", targetNodeName, err)
	}

	peerInfo := new(peer.AddrInfo)
	dataType, err := json.Marshal(destInfo)
	if err != nil {
		return nil, fmt.Errorf("Marshal addr err: %v", err)
	}
	err = peerInfo.UnmarshalJSON(dataType)
	if err != nil {
		return nil, fmt.Errorf("UnmarshalJSON addr %s err: %v", targetNodeName, err)
	}

	connNum := tp.host.Network().ConnsToPeer(peerInfo.ID)
	if len(connNum) >= 2 {
		klog.V(4).Infof("Data transfer between %s is p2p mode", targetNodeName)
	} else {
		klog.V(4).Infof("Try to hole punch with %s", targetNodeName)
		err = tp.host.Connect(context.Background(), *peerInfo)
		if err != nil {
			return nil, fmt.Errorf("connect to %s err: %v", targetNodeName, err)
		}
		klog.V(4).Infof("Data transfer between %s is p2p mode", targetNodeName)
	}

	stream, err := tp.host.NewStream(context.Background(), peerInfo.ID, TCPProxyProtocol)
	if err != nil {
		return nil, fmt.Errorf("New stream between %s err: %v", targetNodeName, err)
	}

	streamWriter := protoio.NewDelimitedWriter(stream)
	streamReader := protoio.NewDelimitedReader(stream, 4*1024)

	msg := &tcp_pb.TCPProxy{
		Type:     tcp_pb.TCPProxy_CONNECT.Enum(),
		Nodename: &targetNodeName,
		Ip:       &targetIP,
		Port:     &targetPort,
	}

	if err = streamWriter.WriteMsg(msg); err != nil {
		err = stream.Reset()
		if err != nil {
			return nil, fmt.Errorf("Stream between %s reset err: %v", targetNodeName, err)
		}
		return nil, fmt.Errorf("Write conn msg to %s err: %v", targetNodeName, err)
	}

	msg.Reset()
	if err = streamReader.ReadMsg(msg); err != nil {
		err = stream.Reset()
		if err != nil {
			return nil, fmt.Errorf("Stream between %s reset err: %v", targetNodeName, err)
		}
		return nil, fmt.Errorf("Read conn result msg from %s err: %v", targetNodeName, err)
	}
	if msg.GetType() == tcp_pb.TCPProxy_FAILED {
		err = stream.Reset()
		if err != nil {
			return nil, fmt.Errorf("Stream between %s reset err: %v", targetNodeName, err)
		}
		return nil, fmt.Errorf("%s dial %s:%d err: %v", targetNodeName, targetIP, targetPort, err)
	}

	msg.Reset()
	klog.Infof("%s dial %s:%d success", targetNodeName, targetIP, targetPort)
	return stream, nil
}
