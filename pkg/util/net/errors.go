package util

import (
	"net"
	"strings"

	"k8s.io/klog/v2"
)

func IsTooManyFDsError(err error) bool {
	return strings.Contains(err.Error(), "too many open files")
}

func IsClosedError(err error) bool {
	return strings.HasSuffix(err.Error(), "use of closed network connection")
}

func IsStreamResetError(err error) bool {
	return strings.HasSuffix(err.Error(), "stream reset")
}

func IsEOFError(err error) bool {
	return strings.HasSuffix(err.Error(), "EOF")
}

func IsTimeoutError(err error) bool {
	if e, ok := err.(net.Error); ok {
		if e.Timeout() {
			klog.V(3).InfoS("Connection to endpoint closed due to inactivity")
			return true
		}
	}
	return false
}
