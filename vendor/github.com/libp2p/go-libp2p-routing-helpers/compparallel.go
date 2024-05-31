package routinghelpers

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/routing"
	"github.com/multiformats/go-multihash"
)

var _ routing.Routing = &composableParallel{}
var _ ProvideManyRouter = &composableParallel{}

type composableParallel struct {
	routers []*ParallelRouter
}

// NewComposableParallel creates a Router that will execute methods from provided Routers in parallel.
// On all methods, If IgnoreError flag is set, that Router will not stop the entire execution.
// On all methods, If ExecuteAfter is set, that Router will be executed after the timer.
// Router specific timeout will start counting AFTER the ExecuteAfter timer.
func NewComposableParallel(routers []*ParallelRouter) *composableParallel {
	return &composableParallel{
		routers: routers,
	}
}

// Provide will call all Routers in parallel.
func (r *composableParallel) Provide(ctx context.Context, cid cid.Cid, provide bool) error {
	return executeParallel(ctx, r.routers,
		func(ctx context.Context, r routing.Routing) error {
			return r.Provide(ctx, cid, provide)
		},
	)
}

// ProvideMany will call all supported Routers in parallel.
func (r *composableParallel) ProvideMany(ctx context.Context, keys []multihash.Multihash) error {
	return executeParallel(ctx, r.routers,
		func(ctx context.Context, r routing.Routing) error {
			pm, ok := r.(ProvideManyRouter)
			if !ok {
				return nil
			}
			return pm.ProvideMany(ctx, keys)
		},
	)
}

// Ready will call all supported ProvideMany Routers SEQUENTIALLY.
// If some of them are not ready, this method will return false.
func (r *composableParallel) Ready() bool {
	for _, ro := range r.routers {
		pm, ok := ro.Router.(ProvideManyRouter)
		if !ok {
			continue
		}

		if !pm.Ready() {
			return false
		}
	}

	return true
}

// FindProvidersAsync will execute all Routers in parallel, iterating results from them in unspecified order.
// If count is set, only that amount of elements will be returned without any specification about from what router is obtained.
// To gather providers from a set of Routers first, you can use the ExecuteAfter timer to delay some Router execution.
func (r *composableParallel) FindProvidersAsync(ctx context.Context, cid cid.Cid, count int) <-chan peer.AddrInfo {
	var totalCount int64
	ch, _ := getChannelOrErrorParallel(
		ctx,
		r.routers,
		func(ctx context.Context, r routing.Routing) (<-chan peer.AddrInfo, error) {
			return r.FindProvidersAsync(ctx, cid, count), nil
		},
		func() bool {
			return atomic.AddInt64(&totalCount, 1) > int64(count) && count != 0
		},
	)

	return ch
}

// FindPeer will execute all Routers in parallel, getting the first AddrInfo found and cancelling all other Router calls.
func (r *composableParallel) FindPeer(ctx context.Context, id peer.ID) (peer.AddrInfo, error) {
	return getValueOrErrorParallel(ctx, r.routers,
		func(ctx context.Context, r routing.Routing) (peer.AddrInfo, bool, error) {
			addr, err := r.FindPeer(ctx, id)
			return addr, addr.ID == "", err
		},
	)
}

// PutValue will execute all Routers in parallel. If a Router fails and IgnoreError flag is not set, the whole execution will fail.
// Some Puts before the failure might be successful, even if we return an error.
func (r *composableParallel) PutValue(ctx context.Context, key string, val []byte, opts ...routing.Option) error {
	return executeParallel(ctx, r.routers,
		func(ctx context.Context, r routing.Routing) error {
			return r.PutValue(ctx, key, val, opts...)
		},
	)
}

// GetValue will execute all Routers in parallel. The first value found will be returned, cancelling all other executions.
func (r *composableParallel) GetValue(ctx context.Context, key string, opts ...routing.Option) ([]byte, error) {
	return getValueOrErrorParallel(ctx, r.routers,
		func(ctx context.Context, r routing.Routing) ([]byte, bool, error) {
			val, err := r.GetValue(ctx, key, opts...)
			return val, len(val) == 0, err
		})
}

func (r *composableParallel) SearchValue(ctx context.Context, key string, opts ...routing.Option) (<-chan []byte, error) {
	return getChannelOrErrorParallel(
		ctx,
		r.routers,
		func(ctx context.Context, r routing.Routing) (<-chan []byte, error) {
			return r.SearchValue(ctx, key, opts...)
		},
		func() bool { return false },
	)
}

func (r *composableParallel) Bootstrap(ctx context.Context) error {
	return executeParallel(ctx, r.routers,
		func(ctx context.Context, r routing.Routing) error {
			return r.Bootstrap(ctx)
		})
}

func getValueOrErrorParallel[T any](
	ctx context.Context,
	routers []*ParallelRouter,
	f func(context.Context, routing.Routing) (T, bool, error),
) (value T, err error) {
	outCh := make(chan T)
	errCh := make(chan error)

	// global cancel context to stop early other router's execution.
	ctx, cancelAll := context.WithCancel(ctx)
	defer cancelAll()
	var wg sync.WaitGroup
	for _, r := range routers {
		wg.Add(1)
		go func(r *ParallelRouter) {
			defer wg.Done()
			tim := time.NewTimer(r.ExecuteAfter)
			defer tim.Stop()
			select {
			case <-ctx.Done():
			case <-tim.C:
				ctx, cancel := context.WithTimeout(ctx, r.Timeout)
				defer cancel()
				value, empty, err := f(ctx, r.Router)
				if err != nil &&
					!errors.Is(err, routing.ErrNotFound) &&
					!r.IgnoreError {
					select {
					case <-ctx.Done():
					case errCh <- err:
					}
					return
				}
				if empty {
					return
				}
				select {
				case <-ctx.Done():
					return
				case outCh <- value:
				}
			}
		}(r)
	}

	// goroutine closing everything when finishing execution
	go func() {
		wg.Wait()
		close(outCh)
		close(errCh)
	}()

	select {
	case out, ok := <-outCh:
		if !ok {
			return value, routing.ErrNotFound
		}
		return out, nil
	case err, ok := <-errCh:
		if !ok {
			return value, routing.ErrNotFound
		}
		return value, err
	case <-ctx.Done():
		return value, ctx.Err()
	}
}

func executeParallel(
	ctx context.Context,
	routers []*ParallelRouter,
	f func(context.Context, routing.Routing,
	) error) error {
	var wg sync.WaitGroup
	errCh := make(chan error)
	for _, r := range routers {
		wg.Add(1)
		go func(r *ParallelRouter) {
			defer wg.Done()
			tim := time.NewTimer(r.ExecuteAfter)
			defer tim.Stop()
			select {
			case <-ctx.Done():
				if !r.IgnoreError {
					errCh <- ctx.Err()
				}
			case <-tim.C:
				ctx, cancel := context.WithTimeout(ctx, r.Timeout)
				defer cancel()
				err := f(ctx, r.Router)
				if err != nil &&
					!r.IgnoreError {
					errCh <- err
				}
			}
		}(r)
	}

	go func() {
		wg.Wait()
		close(errCh)
	}()

	var errOut error
	for err := range errCh {
		errOut = multierror.Append(errOut, err)
	}

	return errOut
}

func getChannelOrErrorParallel[T any](
	ctx context.Context,
	routers []*ParallelRouter,
	f func(context.Context, routing.Routing) (<-chan T, error),
	shouldStop func() bool,
) (chan T, error) {
	outCh := make(chan T)
	errCh := make(chan error)
	var wg sync.WaitGroup
	ctx, cancelAll := context.WithCancel(ctx)
	for _, r := range routers {
		wg.Add(1)
		go func(r *ParallelRouter) {
			defer wg.Done()
			tim := time.NewTimer(r.ExecuteAfter)
			defer tim.Stop()
			select {
			case <-ctx.Done():
				return
			case <-tim.C:
				ctx, cancel := context.WithTimeout(ctx, r.Timeout)
				defer cancel()
				valueChan, err := f(ctx, r.Router)
				if err != nil && !r.IgnoreError {
					select {
					case <-ctx.Done():
					case errCh <- err:
					}
					return
				}
				for {
					select {
					case <-ctx.Done():
						return
					case val, ok := <-valueChan:
						if !ok {
							return
						}

						if shouldStop() {
							return
						}

						select {
						case <-ctx.Done():
							return
						case outCh <- val:
						}
					}
				}
			}
		}(r)
	}

	// goroutine closing everything when finishing execution
	go func() {
		wg.Wait()
		close(outCh)
		close(errCh)
		cancelAll()
	}()

	select {
	case err, ok := <-errCh:
		if !ok {
			return nil, routing.ErrNotFound
		}
		return nil, err
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		return outCh, nil
	}
}
