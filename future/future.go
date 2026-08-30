package future

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/hq-cml/go-tools/gtx"
	"github.com/hq-cml/go-tools/recover"
)

// Value 允许返回任意类型的结果
type Value interface{}

// FutureIntfs Future 接口定义
type FutureIntfs interface {
	// 阻塞等待异步结果返回
	Get() (Value, error)
	// 阻塞等待，但超过 timeout 时长后返回 ErrTimeout；超时后异步任务仍在后台继续执行
	GetWithTimeout(timeout time.Duration) (Value, error)
	// 阻塞等待，同时受 context.Context 控制；当 Context 被取消或到 deadline 时立即返回；但异步任务本身不受影响，继续执行
	GetWithContext(context.Context) (Value, error)
}

// Returned when a Future has timed out
var ErrTimeout = context.DeadlineExceeded

// Returned when a Future has be canceled
var ErrCanceled = context.Canceled

// Returned when a Future has been consumed by a previous Get call
var ErrEmpty = errors.New("empty result")

type Future struct {
	result chan *result
}

type result struct {
	value Value
	err   error
}

// NewFuture 创建并启动一个独立的 Future。fc 将被异步调用，
// 通过返回的 FutureIntfs 的 Get/GetWithTimeout/GetWithContext 获取结果。
// 可通过 opts 传入 WithLogger、WithGtxKeys 等选项。
//
// 注意：Future 的结果只可被消费一次，多次或并发调用 Get 时仅有一个调用方能
// 拿到结果，其余将返回 ErrEmpty。
func NewFuture(fc func() (Value, error), opts ...Option) FutureIntfs {
	return startFuture(fc, nil, newConfig(opts))
}

func (f *Future) Get() (Value, error) {
	return f.GetWithContext(context.Background())
}

func (f *Future) GetWithTimeout(timeout time.Duration) (Value, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return f.GetWithContext(ctx)
}

func (f *Future) GetWithContext(ctx context.Context) (Value, error) {
	// 结果已就绪时优先返回，避免结果与 ctx.Done 同时就绪时被 select 随机丢弃
	select {
	case ret := <-f.result:
		return f.unpack(ret)
	default:
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case ret := <-f.result:
		return f.unpack(ret)
	}
}

func (f *Future) unpack(ret *result) (Value, error) {
	if ret == nil {
		return nil, ErrEmpty
	}
	return ret.value, ret.err
}

// 底层内核
// startFuture 创建 Future 并启动 goroutine 异步执行 fc，统一处理 gtx 上下文传递、panic 恢复与结果写入。
// onExit 作为 goroutine 最外层的 defer 注册（LIFO 最后执行），用于注入额外清理逻辑（如 pool 的计数/信号量释放），可为 nil。
// cfg 控制 gtx 传递、日志等行为，nil 时使用默认配置。
func startFuture(fc func() (Value, error), onExit func(), cfg *futureCFG) *Future {
	if cfg == nil {
		cfg = newDefaultConfig()
	}

	// 从父 goroutine 读取需要传递的 gtx 值
	type kv struct {
		key   interface{}
		value interface{}
		exist bool
	}
	var pairs []kv
	for key := range cfg.gtxKeys {
		v, ok := gtx.Get(key)
		pairs = append(pairs, kv{key, v, ok})
	}

	f := &Future{
		result: make(chan *result, 1),
	}
	var value Value
	var err error
	go func() {
		// onExit 最先注册、最后执行，确保 result 已写入并关闭后再做 pool 清理
		if onExit != nil {
			defer onExit()
		}

		// 初始化当前 goroutine 的 gtx
		gtx.Init4Current()
		defer gtx.Clear4Current()

		// 传递 gtx 上下文（含 LogId/LaneTag 及自定义 keys）
		for _, p := range pairs {
			if p.exist {
				gtx.Set(p.key, p.value)
			}
		}

		// 注册收尾：记录耗时、写入结果并关闭 channel
		startTime := time.Now()
		defer func() {
			cost := time.Since(startTime)
			cfg.logger.Printf("future cost: %v", cost)
			f.result <- &result{value, err}
			close(f.result)
		}()

		// 实际执行 & 附带异常恢复
		// 注意：panic 会被 recover.WithRecover 包装为带堆栈的字符串错误，
		// 原始 panic 值的类型将丢失，调用方无法用 errors.Is/As 判断底层错误。
		panicErr := recover.WithRecover(func() {
			value, err = fc()
		}, nil)
		if panicErr != nil {
			err = fmt.Errorf("panic recover:%v", panicErr)
		}
	}()

	return f
}
