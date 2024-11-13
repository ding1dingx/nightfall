package worker

import (
	"context"
	"runtime/debug"
	"sync"
	"time"

	"github.com/shenghui0779/nightfall/linklist"
)

const (
	defaultPoolCap     = 10000
	defaultIdleTimeout = 60 * time.Second
)

// Pool 协程池，控制并发协程数量，降低CPU和内存负载
type Pool interface {
	// Go 异步执行任务
	Go(context.Context, func(ctx context.Context))

	// Close 关闭资源
	Close()
}

// PanicFn 处理Panic方法
type PanicFn func(ctx context.Context, err any, stack []byte)

type worker struct {
	timeUsed time.Time
	cancel   context.CancelFunc
}

type task struct {
	ctx context.Context
	fn  func(ctx context.Context)
}

type pool struct {
	input chan *task
	queue chan *task
	cache *linklist.DoublyLinkList[*task]

	capacity int
	workers  *linklist.DoublyLinkList[*worker]

	prefill     int
	nonBlock    bool
	queueCap    int
	idleTimeout time.Duration
	panicFn     PanicFn

	ctx    context.Context
	cancel context.CancelFunc
}

// NewPool 生成一个新的Pool
func NewPool(cap int, opts ...Option) Pool {
	if cap <= 0 {
		cap = defaultPoolCap
	}

	ctx, cancel := context.WithCancel(context.TODO())
	p := &pool{
		input: make(chan *task),
		cache: linklist.New[*task](),

		capacity: cap,
		workers:  linklist.New[*worker](),

		idleTimeout: defaultIdleTimeout,

		ctx:    ctx,
		cancel: cancel,
	}

	for _, fn := range opts {
		fn(p)
	}
	p.queue = make(chan *task, p.queueCap)
	// 预填充
	if p.prefill > 0 {
		count := p.prefill
		if p.prefill > p.capacity {
			count = p.capacity
		}
		for i := 0; i < count; i++ {
			p.spawn()
		}
	}

	go p.run()
	go p.idleCheck()

	return p
}

func (p *pool) Go(ctx context.Context, fn func(ctx context.Context)) {
	p.input <- &task{ctx: ctx, fn: fn}
}

func (p *pool) Close() {
	// 销毁协程
	p.cancel()
	time.Sleep(time.Second)
	// 关闭通道
	close(p.input)
	close(p.queue)
}

func (p *pool) run() {
	for {
		select {
		case <-p.ctx.Done():
			return
		case t := <-p.input:
			select {
			case p.queue <- t:
			default:
				if p.workers.Size() < p.capacity {
					// 新开一个协程
					p.spawn()
				}
				if p.nonBlock {
					// 非阻塞模式，放入本地缓存
					p.cache.Append(t)
				} else {
					// 阻塞模式，等待闲置协程
					select {
					case <-p.ctx.Done():
						return
					case p.queue <- t:
					}
				}
			}
		}
	}
}

func (p *pool) idleCheck() {
	ticker := time.NewTicker(p.idleTimeout)
	defer ticker.Stop()

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			idles := p.workers.Filter(func(index int, value *worker) bool {
				if time.Since(value.timeUsed) > p.idleTimeout {
					return true
				}
				return false
			})
			for _, wk := range idles {
				wk.cancel()
			}
		}
	}
}

func (p *pool) spawn() {
	ctx, cancel := context.WithCancel(context.TODO())
	wk := &worker{
		timeUsed: time.Now(),
		cancel:   cancel,
	}
	// 存储协程信息
	p.workers.Append(wk)

	go func(ctx context.Context, wk *worker) {
		var taskCtx context.Context
		defer func() {
			if e := recover(); e != nil {
				if p.panicFn != nil {
					p.panicFn(taskCtx, e, debug.Stack())
				}
			}
		}()
		for {
			var t *task
			// 获取任务
			select {
			case <-p.ctx.Done(): // Pool关闭，销毁
				return
			case <-ctx.Done(): // 闲置超时，销毁
				return
			case t = <-p.queue:
			default:
				// 非阻塞模式，去取缓存的任务执行
				if p.nonBlock {
					t, _ = p.cache.Remove(0)
					if t != nil {
						break
					}
				}
				// 阻塞模式或未取到缓存任务，则等待新任务
				select {
				case <-p.ctx.Done():
					return
				case <-ctx.Done():
					return
				case t = <-p.queue:
				}
			}
			// 执行任务
			wk.timeUsed = time.Now()
			taskCtx = t.ctx
			t.fn(t.ctx)
		}
	}(ctx, wk)
}

var (
	pp   Pool
	once sync.Once
)

// Init 初始化默认的全局Pool
func Init(cap int, opts ...Option) {
	pp = NewPool(cap, opts...)
}

// P 返回默认的全局Pool
func P() Pool {
	if pp == nil {
		once.Do(func() {
			pp = NewPool(defaultPoolCap)
		})
	}
	return pp
}
