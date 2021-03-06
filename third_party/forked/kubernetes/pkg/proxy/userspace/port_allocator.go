// This package is copied from Kubernetes project.
// https://github.com/kubernetes/kubernetes/blob/v1.23.0/pkg/proxy/userspace/port_allocator.go
// The functions is not exposed
package userspace

import (
	"errors"
	"math/big"
	"math/rand"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/util/net"
	"k8s.io/apimachinery/pkg/util/wait"
)

var (
	errPortRangeNoPortsRemaining = errors.New("port allocation failed; there are no remaining ports left to allocate in the accepted range")
)

type PortAllocator interface {
	AllocateNext() (int, error)
	Release(int)
}

// randomAllocator is a PortAllocator implementation that allocates random ports, yielding
// a port value of 0 for every call to AllocateNext().
type randomAllocator struct{}

// AllocateNext always returns 0
func (r *randomAllocator) AllocateNext() (int, error) {
	return 0, nil
}

// Release is a noop
func (r *randomAllocator) Release(_ int) {
	// noop
}

// newPortAllocator builds PortAllocator for a given PortRange. If the PortRange is empty
// then a random port allocator is returned; otherwise, a new range-based allocator
// is returned.
func newPortAllocator(r net.PortRange) PortAllocator {
	if r.Base == 0 {
		return &randomAllocator{}
	}
	return newPortRangeAllocator(r, true)
}

const (
	portsBufSize         = 16
	nextFreePortCooldown = 500 * time.Millisecond
	allocateNextTimeout  = 1 * time.Second
)

type rangeAllocator struct {
	net.PortRange
	ports chan int
	used  big.Int
	lock  sync.Mutex
	rand  *rand.Rand
}

func newPortRangeAllocator(r net.PortRange, autoFill bool) PortAllocator {
	if r.Base == 0 || r.Size == 0 {
		panic("illegal argument: may not specify an empty port range")
	}
	ra := &rangeAllocator{
		PortRange: r,
		ports:     make(chan int, portsBufSize),
		rand:      rand.New(rand.NewSource(time.Now().UnixNano())),
	}
	if autoFill {
		go wait.Forever(func() { ra.fillPorts() }, nextFreePortCooldown)
	}
	return ra
}

// fillPorts loops, always searching for the next free port and, if found, fills the ports buffer with it.
// this func blocks unless there are no remaining free ports.
func (r *rangeAllocator) fillPorts() {
	for {
		if !r.fillPortsOnce() {
			return
		}
	}
}

func (r *rangeAllocator) fillPortsOnce() bool {
	port := r.nextFreePort()
	if port == -1 {
		return false
	}
	r.ports <- port
	return true
}

// nextFreePort finds a free port, first picking a random port. if that port is already in use
// then the port range is scanned sequentially until either a port is found or the scan completes
// unsuccessfully. an unsuccessful scan returns a port of -1.
func (r *rangeAllocator) nextFreePort() int {
	r.lock.Lock()
	defer r.lock.Unlock()

	// choose random port
	j := r.rand.Intn(r.Size)
	if b := r.used.Bit(j); b == 0 {
		r.used.SetBit(&r.used, j, 1)
		return j + r.Base
	}

	// search sequentially
	for i := j + 1; i < r.Size; i++ {
		if b := r.used.Bit(i); b == 0 {
			r.used.SetBit(&r.used, i, 1)
			return i + r.Base
		}
	}
	for i := 0; i < j; i++ {
		if b := r.used.Bit(i); b == 0 {
			r.used.SetBit(&r.used, i, 1)
			return i + r.Base
		}
	}
	return -1
}

func (r *rangeAllocator) AllocateNext() (port int, err error) {
	select {
	case port = <-r.ports:
	case <-time.After(allocateNextTimeout):
		err = errPortRangeNoPortsRemaining
	}
	return
}

func (r *rangeAllocator) Release(port int) {
	port -= r.Base
	if port < 0 || port >= r.Size {
		return
	}
	r.lock.Lock()
	defer r.lock.Unlock()
	r.used.SetBit(&r.used, port, 0)
}
