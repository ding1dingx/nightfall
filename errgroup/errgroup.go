package errgroup

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"

	"github.com/shenghui0779/nightfall/goworker"
)

// A group is a collection of goroutines working on subtasks that are part of
// the same overall task.
//
// A zero group is valid, has no limit on the number of active goroutines,
// and does not cancel on error. use WithContext instead.
type Group interface {
	// Go calls the given function in a new goroutine.
	//
	// The first call to return a non-nil error cancels the group; its error will be
	// returned by Wait.
	Go(fn func(ctx context.Context) error)

	// GOMAXPROCS set max goroutine to work.
	GOMAXPROCS(int)

	// Wait blocks until all function calls from the Go method have returned, then
	// returns the first non-nil error (if any) from them.
	Wait() error
}

type group struct {
	initOnce sync.Once

	wg      sync.WaitGroup
	pool    goworker.Pool
	errOnce sync.Once
	err     error

	workOnce sync.Once
	ch       chan func(ctx context.Context) error
	cache    []func(ctx context.Context) error

	ctx    context.Context
	cancel context.CancelCauseFunc
}

// WithContext returns a new group with a canceled Context derived from ctx.
//
// The derived Context is canceled the first time a function passed to Go
// returns a non-nil error or the first time Wait returns, whichever occurs first.
func WithContext(ctx context.Context, pool goworker.Pool) Group {
	if ctx == nil {
		ctx = context.Background()
	}
	if pool == nil {
		pool = goworker.P()
	}
	ctx, cancel := context.WithCancelCause(ctx)
	return &group{pool: pool, ctx: ctx, cancel: cancel}
}

func (g *group) GOMAXPROCS(n int) {
	if n <= 0 {
		return
	}
	g.workOnce.Do(func() {
		g.ch = make(chan func(context.Context) error, n)
		for i := 0; i < n; i++ {
			g.pool.Go(g.ctx, func(ctx context.Context) {
				for fn := range g.ch {
					g.do(ctx, fn)
				}
			})
		}
	})
}

func (g *group) Go(fn func(ctx context.Context) error) {
	g.wg.Add(1)
	if g.ch != nil {
		select {
		case g.ch <- fn:
		default:
			g.cache = append(g.cache, fn)
		}
		return
	}
	g.pool.Go(g.ctx, func(ctx context.Context) {
		g.do(ctx, fn)
	})
}

func (g *group) Wait() error {
	if g.ch != nil {
		for _, fn := range g.cache {
			g.ch <- fn
		}
	}
	g.wg.Wait()
	if g.ch != nil {
		close(g.ch) // let all receiver exit
	}
	if g.cancel != nil {
		g.cancel(g.err)
	}
	return g.err
}

func (g *group) do(ctx context.Context, fn func(ctx context.Context) error) {
	var err error
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("errgroup panic recovered: %+v\n%s", r, string(debug.Stack()))
		}
		if err != nil {
			// fmt.Println(err)
			g.errOnce.Do(func() {
				g.err = err
				if g.cancel != nil {
					g.cancel(err)
				}
			})
		}
		g.wg.Done()
	}()
	err = fn(ctx)
}
