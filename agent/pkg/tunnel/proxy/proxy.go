package proxy

import (
	"context"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/libp2p/go-msgio/protoio"
	"k8s.io/klog/v2"

	"github.com/kubeedge/edgemesh/agent/pkg/tunnel/controller"
	"github.com/kubeedge/edgemesh/agent/pkg/tunnel/proxy/pb"
	"github.com/kubeedge/edgemesh/common/util"
)

const (
	MaxReadSize  = 4096
	MaxRetryTime = 3
)

var ProxyProtocol protocol.ID = "/libp2p/tunnel-proxy/1.0.0"

type ProxyService struct {
	host host.Host
}

func NewProxyService(h host.Host) *ProxyService {
	return &ProxyService{
		host: h,
	}
}

type ProxyOptions struct {
	Protocol string
	NodeName string
	IP       string
	Port     int32
}

func (ps *ProxyService) ProxyStreamHandler(stream network.Stream) {
	// todo use peerID to get nodeName
	klog.Infof("Got a new stream from %s", stream.Conn().RemotePeer().String())
	streamWriter := protoio.NewDelimitedWriter(stream)
	streamReader := protoio.NewDelimitedReader(stream, MaxReadSize)

	msg := new(pb.Proxy)
	err := streamReader.ReadMsg(msg)
	if err != nil {
		klog.Errorf("Read msg from %s err: %v", stream.Conn().RemotePeer().String(), err)
		return
	}
	if msg.GetType() != pb.Proxy_CONNECT {
		klog.Errorf("Read msg from %s type should be CONNECT", stream.Conn().RemotePeer().String())
		return
	}
	targetProto := msg.GetProtocol()
	targetNode := msg.GetNodeName()
	targetAddr := fmt.Sprintf("%s:%d", msg.GetIp(), msg.GetPort())

	proxyConn, err := ps.TryConnectEndpoint(msg)
	if err != nil {
		klog.Errorf("l4 proxy connect to %v err: %v", msg, err)
		msg.Reset()
		msg.Type = pb.Proxy_FAILED.Enum()
		if err = streamWriter.WriteMsg(msg); err != nil {
			klog.Errorf("Write msg to %s err: %v", stream.Conn().RemotePeer().String(), err)
			return
		}
		return
	}

	msg.Reset()
	msg.Type = pb.Proxy_SUCCESS.Enum()
	if err = streamWriter.WriteMsg(msg); err != nil {
		klog.Errorf("Write msg to %s err: %v", stream.Conn().RemotePeer().String(), err)
		return
	}
	msg.Reset()

	switch targetProto {
	case "tcp":
		go util.ProxyStream(stream, proxyConn)
	case "udp":
		go util.ProxyStreamUDP(stream, proxyConn.(*net.UDPConn))
	}
	klog.Infof("Success proxy for {%s %s %s}", targetProto, targetNode, targetAddr)
}

func (ps *ProxyService) TryConnectEndpoint(msg *pb.Proxy) (net.Conn, error) {
	var err error
	switch msg.GetProtocol() {
	case "tcp":
		for i := 0; i < MaxRetryTime; i++ {
			tcpConn, err := net.DialTCP("tcp", nil, &net.TCPAddr{
				IP:   net.ParseIP(msg.GetIp()),
				Port: int(msg.GetPort()),
			})
			if err == nil {
				return tcpConn, nil
			}
			time.Sleep(time.Second)
		}
		klog.Errorf("max retries for dial")
		return nil, err
	case "udp":
		for i := 0; i < MaxRetryTime; i++ {
			udpConn, err := net.DialUDP("udp", nil, &net.UDPAddr{
				IP:   net.ParseIP(msg.GetIp()),
				Port: int(msg.GetPort()),
			})
			if err == nil {
				return udpConn, nil
			}
		}
		klog.Errorf("max retries for dial")
		return nil, err
	default:
		return nil, fmt.Errorf("unsupported protocol: %s", msg.GetProtocol())
	}
}

func (ps *ProxyService) GetProxyStream(opts ProxyOptions) (io.ReadWriteCloser, error) {
	destInfo, err := controller.APIConn.GetPeerAddrInfo(opts.NodeName)
	if err != nil {
		return nil, fmt.Errorf("get %s addr err: %w", opts.NodeName, err)
	}

	connNum := ps.host.Network().ConnsToPeer(destInfo.ID)
	if len(connNum) >= 2 {
		klog.V(4).Infof("Data transfer between %s is p2p mode", opts.NodeName)
	} else {
		klog.V(4).Infof("Try to hole punch with %s", opts.NodeName)
		err = ps.host.Connect(context.Background(), *destInfo)
		if err != nil {
			return nil, fmt.Errorf("connect to %s err: %v", opts.NodeName, err)
		}
		klog.V(4).Infof("Data transfer between %s is p2p mode", opts.NodeName)
	}

	stream, err := ps.host.NewStream(context.Background(), destInfo.ID, ProxyProtocol)
	if err != nil {
		return nil, fmt.Errorf("new stream between %s err: %w", opts.NodeName, err)
	}

	streamWriter := protoio.NewDelimitedWriter(stream)
	streamReader := protoio.NewDelimitedReader(stream, MaxReadSize)

	msg := &pb.Proxy{
		Type:     pb.Proxy_CONNECT.Enum(),
		Protocol: &opts.Protocol,
		NodeName: &opts.NodeName,
		Ip:       &opts.IP,
		Port:     &opts.Port,
	}

	if err = streamWriter.WriteMsg(msg); err != nil {
		resetErr := stream.Reset()
		if resetErr != nil {
			return nil, fmt.Errorf("stream between %s reset err: %w", opts.NodeName, resetErr)
		}
		return nil, fmt.Errorf("write conn msg to %s err: %w", opts.NodeName, err)
	}

	msg.Reset()
	if err = streamReader.ReadMsg(msg); err != nil {
		resetErr := stream.Reset()
		if resetErr != nil {
			return nil, fmt.Errorf("stream between %s reset err: %w", opts.NodeName, resetErr)
		}
		return nil, fmt.Errorf("read conn result msg from %s err: %w", opts.NodeName, err)
	}

	if msg.GetType() == pb.Proxy_FAILED {
		resetErr := stream.Reset()
		if resetErr != nil {
			return nil, fmt.Errorf("stream between %s reset err: %w", opts.NodeName, err)
		}
		return nil, fmt.Errorf("libp2p dial %v err: Proxy.type is %s", opts, pb.Proxy_FAILED)
	}

	msg.Reset()
	klog.V(4).Infof("libp2p dial %v success", opts)

	return stream, nil
}
