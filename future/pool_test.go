package future

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestPool_Basic(t *testing.T) {
	pool, err := NewPool(2, 0)
	if err != nil {
		t.Fatalf("NewPool err: %v", err)
	}
	defer pool.Close()

	f, err := pool.Submit(context.Background(), func() (Value, error) {
		return 42, nil
	})
	if err != nil {
		t.Fatalf("Submit err: %v", err)
	}
	val, err := f.Get()
	if err != nil {
		t.Fatalf("Get err: %v", err)
	}
	if val != 42 {
		t.Fatalf("val = %v, want 42", val)
	}
}

func TestPool_InvalidSize(t *testing.T) {
	if _, err := NewPool(0, 0); err == nil {
		t.Fatal("want error for size=0")
	}
	if _, err := NewPool(-1, 0); err == nil {
		t.Fatal("want error for negative size")
	}
}

// 测试并发保证
func TestPool_ConcurrencyLimit(t *testing.T) {
	const size = 3
	pool, err := NewPool(size, 0)
	if err != nil {
		t.Fatalf("NewPool err: %v", err)
	}

	var running int64
	var maxRunning int64 // 历史并发goroutine峰值，改值最大就应该是3，不能再大
	for i := 0; i < 10; i++ {
		if _, err := pool.Submit(context.Background(), func() (Value, error) {
			// r包括当前正在运行的goroutine，一共在并发的goroutine数
			r := atomic.AddInt64(&running, 1)

			// 更新历史并发goroutine峰值，这个值最大就应该是3，不能再大
			// 若 r <= m：自己不是新高，直接跳过；
			// 否则 CAS 尝试把峰值改成 r：若期间别的 goroutine 抢先改成了更大的值，CAS 失败返回false，通过for循环重读再比
			// 可能返回 false 的场景：两个 goroutine 同时读到 m=2，都想 CAS 更新。一个先成功把峰值改成 3，
			// 						另一个 CAS 时发现 *addr(3) != m(2)，返回 false，进入下一次循环重新 Load。
			for {
				m := atomic.LoadInt64(&maxRunning)
				if r <= m || atomic.CompareAndSwapInt64(&maxRunning, m, r) {
					break
				}
			}

			// 模拟实际的任务执行过程
			time.Sleep(20 * time.Millisecond)

			// goroutine执行完毕，数量减回去
			atomic.AddInt64(&running, -1)
			return nil, nil
		}); err != nil {
			t.Fatalf("Submit err: %v", err)
		}
	}

	pool.Close()

	// maxRunning <= 3：因为信号量保证并发 ≤ 3，峰值必然不超 3
	if maxRunning > size {
		t.Fatalf("max running = %d, want <= %d", maxRunning, size)
	}
	// Close 等待所有任务结束后，counter 应归零，验证无 goroutine 泄漏
	if got := pool.GetGoroutineCount(); got != 0 {
		t.Fatalf("counter after Close = %d, want 0", got)
	}
}

func TestPool_SubmitTimeout(t *testing.T) {
	pool, err := NewPool(1, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("NewPool err: %v", err)
	}
	defer pool.Close()

	// 占满唯一的信号量，这个任务会永久阻塞
	blockCh := make(chan struct{})
	defer close(blockCh)
	if _, err := pool.Submit(context.Background(), func() (Value, error) {
		<-blockCh
		return nil, nil
	}); err != nil {
		t.Fatalf("Submit err: %v", err)
	}

	// 第二个任务，提交会超时
	start := time.Now()
	_, err = pool.Submit(context.Background(), func() (Value, error) { return nil, nil })
	fmt.Println(err)
	if !errors.Is(err, ErrPoolTimeout) {
		t.Fatalf("want ErrPoolTimeout, got %v", err)
	}
	if elapsed := time.Since(start); elapsed < 40*time.Millisecond {
		t.Fatalf("returned too fast: %v", elapsed)
	}
}

func TestPool_Closed(t *testing.T) {
	pool, err := NewPool(1, 0)
	if err != nil {
		t.Fatalf("NewPool err: %v", err)
	}
	pool.Close()

	_, err = pool.Submit(context.Background(), func() (Value, error) { return nil, nil })
	if !errors.Is(err, ErrPoolClosed) {
		t.Fatalf("want ErrPoolClosed, got %v", err)
	}
	fmt.Println(err)
}

func TestPool_SubmitContextCanceled(t *testing.T) {
	pool, err := NewPool(1, 0)
	if err != nil {
		t.Fatalf("NewPool err: %v", err)
	}
	defer pool.Close()

	// 永远阻塞的任务
	blockCh := make(chan struct{})
	defer close(blockCh)
	if _, err := pool.Submit(context.Background(), func() (Value, error) {
		<-blockCh
		return nil, nil
	}); err != nil {
		t.Fatalf("Submit err: %v", err)
	}

	// 第二个任务提交，只能等待取消
	ctx, cancel := context.WithCancel(context.Background())
	go func () {
		time.Sleep(1 * time.Second)
		fmt.Println("do cancel")
		cancel()
	}()

	_, err = pool.Submit(ctx, func() (Value, error) { return nil, nil })
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("want context.Canceled, got %v", err)
	}
	fmt.Println(err)
}

func TestPool_SubmitContextDeadline(t *testing.T) {
	pool, err := NewPool(1, 0)
	if err != nil {
		t.Fatalf("NewPool err: %v", err)
	}
	defer pool.Close()

	// 永远阻塞的任务
	blockCh := make(chan struct{})
	defer close(blockCh)
	if _, err := pool.Submit(context.Background(), func() (Value, error) {
		<-blockCh
		return nil, nil
	}); err != nil {
		t.Fatalf("Submit err: %v", err)
	}

	// 第二个任务提交，会超时
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err = pool.Submit(ctx, func() (Value, error) { return nil, nil })
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("want context.DeadlineExceeded, got %v", err)
	}
	fmt.Println(err)
}

func TestPool_CloseWaitsForRunningTasks(t *testing.T) {
	pool, err := NewPool(1, 0)
	if err != nil {
		t.Fatalf("NewPool err: %v", err)
	}

	// 提交任务：任务阻塞在 <-doneCh，不结束（占住唯一的信号量）。
	doneCh := make(chan struct{})
	done := int32(0)
	if _, err := pool.Submit(context.Background(), func() (Value, error) {
		<-doneCh
		atomic.StoreInt32(&done, 1)
		return nil, nil
	}); err != nil {
		t.Fatalf("Submit err: %v", err)
	}

	// 在goroutine中调 Close()：Close 内部 wg.Wait() 会一直等任务结束，完成后才关 closeCh
	closeCh := make(chan struct{})
	go func() {
		pool.Close()
		close(closeCh)
	}()

	// select 判断：50ms 内若 closeCh 已关闭，说明 Close 没等任务就返回了 → 测试失败；反之证明 Close 确实在等待。
	select {
	case <-closeCh:
		t.Fatal("Close returned before task finished")
	case <-time.After(50 * time.Millisecond):
	}

	// 放行任务：close(doneCh) 让任务结束，此时 <-closeCh 才能返回，说明 Close 是在任务完成后才退出的。
	close(doneCh)

	// <-closeCh 才能返回
	<-closeCh
	if atomic.LoadInt32(&done) != 1 {
		t.Fatal("task not finished before Close returned")
	}
}

func TestPool_GetGoroutineCount(t *testing.T) {
	pool, err := NewPool(1, 0)
	if err != nil {
		t.Fatalf("NewPool err: %v", err)
	}
	defer pool.Close()

	blockCh := make(chan struct{})
	released := make(chan struct{})
	if _, err := pool.Submit(context.Background(), func() (Value, error) {
		<-blockCh
		close(released)
		return nil, nil
	}); err != nil {
		t.Fatalf("Submit err: %v", err)
	}

	// 等待任务真正开始运行（信号量已占用）
	for i := 0; i < 100; i++ {
		if pool.GetGoroutineCount() == 1 {
			break
		}
		fmt.Println("F")
		time.Sleep(5 * time.Millisecond)
	}
	if got := pool.GetGoroutineCount(); got != 1 {
		t.Fatalf("counter = %d, want 1", got)
	}

	close(blockCh)
	<-released
	// 等待任务结束后计数归零
	for i := 0; i < 100; i++ {
		if pool.GetGoroutineCount() == 0 {
			break
		}
		fmt.Println("B")
		time.Sleep(5 * time.Millisecond)
	}
	if got := pool.GetGoroutineCount(); got != 0 {
		t.Fatalf("counter = %d, want 0", got)
	}
}

func TestPool_ManyTasks(t *testing.T) {
	pool, err := NewPool(8, 0)
	if err != nil {
		t.Fatalf("NewPool err: %v", err)
	}

	const n = 50
	var wg sync.WaitGroup
	errCh := make(chan error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		f, err := pool.Submit(context.Background(), func() (Value, error) {
			return 7, nil
		})
		if err != nil {
			t.Fatalf("Submit err: %v", err)
		}
		go func(f FutureIntfs) {
			defer wg.Done()
			if v, err := f.Get(); err != nil || v != 7 {
				errCh <- fmt.Errorf("Get = %v, %v; want 7, nil", v, err)
			}
		}(f)
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Error(err)
	}
	pool.Close()
}

// 8 个 goroutine 疯狂提交：每个 goroutine 死循环调 Submit，直到收到 stop 信号才退出，制造持续的并发提交压力。
// 20ms 后主 goroutine 调 Close()：此时可能已有任务在跑、也有 goroutine 正在提交，形成 Submit 与 Close 并发竞争。
// 关 stop + wg.Wait()：等 8 个提交 goroutine 全部退出，测试结束
func TestPool_CloseConcurrent(t *testing.T) {
	pool, err := NewPool(4, 0)
	if err != nil {
		t.Fatalf("NewPool err: %v", err)
	}

	// 并发提交的同时关闭，验证无死锁、无数据竞争
	stop := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
					pool.Submit(context.Background(), func() (Value, error) {
						return nil, nil
					})
				}
			}
		}()
	}

	time.Sleep(20 * time.Millisecond)
	pool.Close()
	close(stop)
	wg.Wait()
}
