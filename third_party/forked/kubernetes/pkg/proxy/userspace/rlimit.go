//go:build !windows
// +build !windows

// This package is copied from Kubernetes project.
// https://github.com/kubernetes/kubernetes/blob/v1.23.0/pkg/proxy/userspace/rlimit.go
// The setRLimit function is not exposed
package userspace

import "golang.org/x/sys/unix"

func setRLimit(limit uint64) error {
	return unix.Setrlimit(unix.RLIMIT_NOFILE, &unix.Rlimit{Max: limit, Cur: limit})
}
