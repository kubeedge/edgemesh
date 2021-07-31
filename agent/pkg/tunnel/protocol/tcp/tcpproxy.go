package tcp

import (
	"context"
	"fmt"
	"github.com/kubeedge/edgemesh/agent/pkg/tunnel/controller"
	"io"
	"net"
	"sync"
	"time"

	tcp_pb "github.com/kubeedge/edgemesh/agent/pkg/tunnel/protocol/tcp/pb"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/libp2p/go-msgio/protoio"
	"k8s.io/klog/v2"
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
	defer s.Reset()
	// todo use peerID to get nodeName
	klog.Infof("Get a new stream from %s", s.Conn().RemotePeer().String())
	streamWriter := protoio.NewDelimitedWriter(s)
	streamReader := protoio.NewDelimitedReader(s, 4*1024)

	msg := new(tcp_pb.TCPProxy)
	if err := streamReader.ReadMsg(msg); err != nil {
		klog.Errorf("Read msg from %s err: err", s.Conn().RemotePeer().String(), err)
		return
	}
	if msg.GetType() != tcp_pb.TCPProxy_CONNECT {
		klog.Errorf("Read msg from %s type should be CONNECT", s.Conn().RemotePeer().String())
		return
	}

	targetIP := msg.GetIp()
	targetPort := msg.GetPort()
	targetAddr := &net.TCPAddr{
		IP:   net.ParseIP(targetIP),
		Port: int(targetPort),
	}
	klog.Infof("l4 proxy get tcp server address: %v", targetAddr)

	var proxyClient net.Conn
	var err error
	// todo retry time use const
	for i := 0; i < 5; i++ {
		proxyClient, err = net.DialTimeout("tcp", targetAddr.String(), 5*time.Second)
		if err == nil {
			break
		}
	}
	defer proxyClient.Close()

	msg.Reset()
	if err != nil {
		klog.Errorf("l4 proxy connect to %s:%d err: %v", targetIP, targetPort, err)
		msg.Type = tcp_pb.TCPProxy_FAILED.Enum()
		if err := streamWriter.WriteMsg(msg); err != nil {
			klog.Errorf("Write msg to %s err: %v", s.Conn().RemotePeer().String(), err)
			return
		}
		return
	}

	msg.Type = tcp_pb.TCPProxy_SUCCESS.Enum()
	if err := streamWriter.WriteMsg(msg); err != nil {
		klog.Errorf("Write msg to %s err: %v", s.Conn().RemotePeer().String(), err)
		return
	}
	msg.Reset()

	wg := sync.WaitGroup{}
	wg.Add(2)

	go func() {
		defer proxyClient.Close()
		defer s.Close()
		_, err = io.Copy(proxyClient, s)
		if err != nil {
			klog.Errorf("io copy error: %v", err)
		}
		wg.Done()
	}()

	go func() {
		defer proxyClient.Close()
		defer s.Close()
		_, err = io.Copy(s, proxyClient)
		if err != nil {
			klog.Errorf("io copy error: %v", err)
		}
		wg.Done()
	}()
	wg.Wait()
	klog.Infof("Success proxy for targetAddr: %v", targetAddr)
}

func (tp *TCPProxyService) GetProxyStream(targetNodeName, targetIP string, targetPort int32) (io.ReadWriteCloser, error) {
	destInfo, err := controller.APIConn.GetPeerAddrInfo(targetNodeName)
	if err != nil {
		return nil, fmt.Errorf("Get %s addr err: %v", targetNodeName, err)
	}

	connNum := tp.host.Network().ConnsToPeer(destInfo.ID)
	if len(connNum) >= 2 {
		klog.V(4).Infof("Data transfer between %s is p2p mode", targetNodeName)
	} else {
		klog.V(4).Infof("Try to hole punch with %s", targetNodeName)
		err = tp.host.Connect(context.Background(), *destInfo)
		if err != nil {
			return nil, fmt.Errorf("connect to %s err: %v", targetNodeName, err)
		}
		klog.V(4).Infof("Data transfer between %s is realy mode", targetNodeName)
	}

	stream, err := tp.host.NewStream(context.Background(), destInfo.ID, TCPProxyProtocol)
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
		stream.Reset()
		return nil, fmt.Errorf("Write conn msg to %s err: %v", targetNodeName, err)
	}
	msg.Reset()

	if err = streamReader.ReadMsg(msg); err != nil {
		stream.Reset()
		return nil, fmt.Errorf("Read conn result msg from %s err: %v", targetNodeName, err)
	}
	if msg.GetType() == tcp_pb.TCPProxy_FAILED {
		stream.Reset()
		return nil, fmt.Errorf("%s dial %s:%d err: %v", targetNodeName, targetIP, targetPort, err)
	}
	msg.Reset()

	klog.Infof("%s dial %s:%d success", targetNodeName, targetIP, targetPort, err)
	return stream, nil
}
