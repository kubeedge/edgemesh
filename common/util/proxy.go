package util

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"k8s.io/klog/v2"
)

// HttpRequestToBytes transforms http.Request to bytes
func HttpRequestToBytes(req *http.Request) ([]byte, error) {
	if req == nil {
		return nil, fmt.Errorf("http request nil")
	}
	buf := new(bytes.Buffer)
	err := req.Write(buf)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Copy and update from https://github.com/kubernetes/kubernetes/blob/v1.23.0/pkg/proxy/userspace/proxysocket.go#L154
// ProxyConn proxies data bi-directionally between in and out.
func ProxyConn(in, out net.Conn) {
	var wg sync.WaitGroup
	wg.Add(2)
	klog.V(4).InfoS("Creating proxy between remote and local addresses",
		"inRemoteAddress", in.RemoteAddr(), "inLocalAddress", in.LocalAddr(), "outLocalAddress", out.LocalAddr(), "outRemoteAddress", out.RemoteAddr())
	go copyBytes("from backend", in, out, &wg)
	go copyBytes("to backend", out, in, &wg)
	wg.Wait()
}

// Copy and update from https://github.com/kubernetes/kubernetes/blob/v1.23.0/pkg/proxy/userspace/proxysocket.go#L164
func copyBytes(direction string, dest, src net.Conn, wg *sync.WaitGroup) {
	defer wg.Done()
	klog.V(4).InfoS("Copying remote address bytes", "direction", direction, "sourceRemoteAddress", src.RemoteAddr(), "destinationRemoteAddress", dest.RemoteAddr())
	n, err := io.Copy(dest, src)
	if err != nil {
		if !IsClosedError(err) && !IsStreamResetError(err) {
			klog.ErrorS(err, "I/O error occurred")
		}
	}
	klog.V(4).InfoS("Copied remote address bytes", "bytes", n, "direction", direction, "sourceRemoteAddress", src.RemoteAddr(), "destinationRemoteAddress", dest.RemoteAddr())
	dest.Close()
	src.Close()
}

func ProxyConnUDP(inConn net.Conn, udpConn *net.UDPConn) {
	var buffer [4096]byte
	for {
		n, err := inConn.Read(buffer[0:])
		if err != nil {
			if e, ok := err.(net.Error); ok {
				if e.Temporary() {
					klog.V(1).ErrorS(err, "ReadFrom had a temporary failure")
					continue
				}
			}
			if !IsClosedError(err) && !IsStreamResetError(err) {
				klog.ErrorS(err, "ReadFrom failed, exiting")
			}
			break
		}
		go copyDatagram(udpConn, inConn)
		_, err = udpConn.Write(buffer[0:n])
		if err != nil {
			if !LogTimeout(err) {
				klog.ErrorS(err, "Write failed")
			}
			continue
		}
		err = udpConn.SetDeadline(time.Now().Add(time.Second))
		if err != nil {
			klog.ErrorS(err, "SetDeadline failed")
			continue
		}
	}
}

func copyDatagram(udpConn *net.UDPConn, outConn net.Conn) {
	defer udpConn.Close()
	var buffer [4096]byte
	for {
		n, _, err := udpConn.ReadFromUDP(buffer[0:])
		if err != nil {
			if !LogTimeout(err) && !IsEOFError(err) {
				klog.ErrorS(err, "Read failed")
			}
			break
		}
		err = udpConn.SetDeadline(time.Now().Add(time.Second))
		if err != nil {
			klog.ErrorS(err, "SetDeadline failed")
			break
		}
		_, err = outConn.Write(buffer[0:n])
		if err != nil {
			if !LogTimeout(err) {
				klog.ErrorS(err, "WriteTo failed")
			}
			break
		}
	}
}

// Copy and paste from https://github.com/kubernetes/kubernetes/blob/v1.23.0/pkg/proxy/userspace/proxier.go#L1259
func IsTooManyFDsError(err error) bool {
	return strings.Contains(err.Error(), "too many open files")
}

// Copy and paste from https://github.com/kubernetes/kubernetes/blob/v1.23.0/pkg/proxy/userspace/proxier.go#L1263
func IsClosedError(err error) bool {
	// A brief discussion about handling closed error here:
	// https://code.google.com/p/go/issues/detail?id=4373#c14
	// TODO: maybe create a stoppable TCP listener that returns a StoppedError
	return strings.HasSuffix(err.Error(), "use of closed network connection")
}

func IsStreamResetError(err error) bool {
	return strings.HasSuffix(err.Error(), "stream reset")
}

func IsEOFError(err error) bool {
	return strings.HasSuffix(err.Error(), "EOF")
}

// Copy and paste from https://github.com/kubernetes/kubernetes/blob/v1.23.0/pkg/proxy/userspace/proxier.go#L111
func LogTimeout(err error) bool {
	if e, ok := err.(net.Error); ok {
		if e.Timeout() {
			klog.V(3).InfoS("Connection to endpoint closed due to inactivity")
			return true
		}
	}
	return false
}
