package future

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// ErrPoolTimeout 在向协程池提交任务时，因信号量满且等待超时而返回。
var ErrPoolTimeout = fmt.Errorf("submit to future goroutine pool timeout")

// ErrPoolClosed 在协程池已关闭后提交任务时返回。
var ErrPoolClosed = fmt.Errorf("future goroutine pool is closed")

type FuturePool struct {
	size    int64
	timeout time.Duration
	cfg     *futureCFG

	mu            sync.Mutex // 保护 closed 读写与 wg.Add，避免与 Close 并发竞态
	semaphoreChan chan bool
	wg            sync.WaitGroup
	counter       int64 // 当前运行的协程数，用 atomic 支持无锁读取
	closed        bool
	closeOnce     sync.Once
}

// NewPool 创建一个指定容量与提交超时时长的协程池。
// size 为允许同时运行的最大 goroutine 数量；timeout 为提交任务时等待空闲
// 信号量的最大时长，<=0 表示不超时（阻塞等待）。
// opts 支持 WithLogger、WithGtxKeys 等选项，对池内所有任务生效。
//
// 警告：池的并发度由信号量限制，token 由任务 goroutine 自身持有，
// 因此任务内部禁止向同一个池再次提交并阻塞等待其结果，否则会因
// 信号量无法释放而造成死锁；同样不要在任务内部调用 Close()（wg.Wait 会
// 等待自身完成，同样死锁）。此限制需由调用方自行规避。
func NewPool(size int64, timeout time.Duration, opts ...Option) (*FuturePool, error) {
	if size <= 0 {
		return nil, fmt.Errorf("pool size must be positive, got %d", size)
	}
	return &FuturePool{
		size:          size,
		timeout:       timeout,
		cfg:           newConfig(opts),
		semaphoreChan: make(chan bool, size),
	}, nil
}

// Submit 向协程池提交一个异步任务，返回 Future 供调用方获取结果。
// ctx 可用于取消阻塞中的信号量等待，当 ctx 被取消或到达 deadline 时返回 ctx.Err()；
// 当池已关闭或提交超时（若设置了 timeout）时返回错误。
//
// 注意：不要在任务内部调用 Submit 提交到同一池并等待结果，否则可能死锁，见 NewPool 的警告。
// 当 timeout <= 0 且池满时，Submit 会一直阻塞等待空闲信号量，可通过 ctx 取消。
func (fp *FuturePool) Submit(ctx context.Context, fc func() (Value, error)) (FutureIntfs, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	// 尝试获取信号量；若达到信号量上限则等待，直到拿到信号量、超时或 ctx 取消
	if err := fp.acquire(ctx); err != nil {
		return nil, err
	}

	// 信号量获取成功后，加锁登记任务
	// 与 Close 的 closed 置位互斥，保证关闭后不再接受新任务、且 wg.Add 不与 wg.Wait 并发
	fp.mu.Lock()
	if fp.closed {
		fp.mu.Unlock()
		<-fp.semaphoreChan // 释放已获取的信号量
		return nil, ErrPoolClosed
	}
	fp.wg.Add(1)
	fp.mu.Unlock()

	// 信号量获取成功，增加协程计数并启动协程
	atomic.AddInt64(&fp.counter, 1)
	return fp.newFutureResult(fc), nil
}

func (fp *FuturePool) acquire(ctx context.Context) error {
	if fp.timeout > 0 {
		timer := time.NewTimer(fp.timeout)
		defer timer.Stop()
		select {
		case fp.semaphoreChan <- true:
			return nil
		case <-timer.C:
			return ErrPoolTimeout
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	select {
	case fp.semaphoreChan <- true:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (fp *FuturePool) GetGoroutineCount() int64 {
	return atomic.LoadInt64(&fp.counter)
}

func (fp *FuturePool) Close() {
	// 注意：不要在池内的任务 goroutine 中调用 Close，wg.Wait 会等待任务自身完成而死锁。
	fp.closeOnce.Do(func() {
		// 在锁内置位 closed，与 Submit 的登记互斥，
		// 确保置位后不再有任务被登记，wg.Wait 不会被后续的 Add 干扰
		fp.mu.Lock()
		fp.closed = true
		fp.mu.Unlock()

		fp.wg.Wait()
	})
}

// newFutureResult 创建有 pool 控制的 FutureResult
func (fp *FuturePool) newFutureResult(fc func() (Value, error)) *Future {
	// onExit 顺序：先减计数、再释放信号量、最后 wg.Done，
	// 保证 Close 的 wg.Wait 返回时计数与信号量均已恢复一致。
	return startFuture(fc, func() {
		atomic.AddInt64(&fp.counter, -1)
		<-fp.semaphoreChan
		fp.wg.Done()
	}, fp.cfg)
}
