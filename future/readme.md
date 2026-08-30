# Future

一个轻量的异步框架：将任务放进 goroutine 异步执行，通过 Future 阻塞获取结果；并提供基于信号量的协程池 `FuturePool`，用于限制并发。

## 特性

- ✓ 异步执行任意 `func() (Value, error)`，支持任意返回类型
- ✓ `Get` / `GetWithTimeout` / `GetWithContext` 三种获取结果方式
- ✓ 任务 panic 自动恢复，不会拖垮整个进程
- ✓ gtx 上下文传递（默认传递 LogId、LaneTag，可自定义扩展）
- ✓ `FuturePool` 信号量限流，可控制最大并发 goroutine 数量
- ✓ `Submit` 支持 context 取消阻塞的信号量等待

## 安装

```bash
go get github.com/hq-cml/go-tools/future
```

## 快速开始

### 1. 单独异步执行一个任务

```go
package main

import (
	"fmt"

	"github.com/hq-cml/go-tools/future"
)

func main() {
	f := future.NewFuture(func() (future.Value, error) {
		// 这里做耗时操作，比如请求下游、查库
		return 42, nil
	})

	// 阻塞等待结果
	val, err := f.Get()
	if err != nil {
		fmt.Println("任务执行失败:", err)
		return
	}
	fmt.Println("结果:", val)
}
```

### 2. 超时等待与 context 控制

```go
// 最多等 1 秒；超时后异步任务仍在后台继续执行
val, err := f.GetWithTimeout(time.Second)

// 由外部 context 控制等待，ctx 被取消/超时时立即返回
ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
defer cancel()
val, err = f.GetWithContext(ctx)
```

### 3. 使用协程池限制并发

```go
// 最多同时运行 4 个 goroutine；提交等待信号量超过 2 秒则返回 ErrPoolTimeout
pool, err := future.NewPool(4, 2*time.Second)
if err != nil {
	panic(err)
}
defer pool.Close() // 等待所有已提交任务执行完成后关闭

for i := 0; i < 100; i++ {
	// 第一个参数 ctx：可用于取消"等待空闲信号量"的阻塞
	// 传 context.Background() 表示一直等下去
	f, err := pool.Submit(context.Background(), func() (future.Value, error) {
		return doSomething(i)
	})
	if err != nil {
		fmt.Println("提交失败:", err)
		continue
	}
	// 拿到 Future，自行决定何时消费结果
	_ = f
}
```

`timeout <= 0` 表示提交时不超时：池满时 `Submit` 会一直阻塞等待空闲信号量，此时可通过传入的 ctx 取消：

```go
ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
defer cancel()
f, err := pool.Submit(ctx, task) // 池满时最多阻塞 3 秒
```

## 选项（Option）

```go
// WithLogger：记录每个 Future 的执行耗时
// 任何实现了 Printf(format, args...) 的对象都可作为 Logger（标准库 *log.Logger 即可）
loggr := log.New(os.Stdout, "", log.LstdFlags)
f := future.NewFuture(fc, future.WithLogger(loggr))

// WithGtxKeys：除默认的 LogId/LaneTag 外，额外传递自定义 gtx key 到子 goroutine
f := future.NewFuture(fc, future.WithGtxKeys("user_id"))
```

子 goroutine 中读取传递的 gtx 值：

```go
if v, ok := gtx.Get("user_id"); ok {
	// ...
}
```

## 错误定义

| 错误 | 说明 |
|------|------|
| `ErrTimeout` | Future 等待超时（与 `context.DeadlineExceeded` 相同） |
| `ErrCanceled` | Future 等待被取消（与 `context.Canceled` 相同） |
| `ErrEmpty` | 结果已被消费过，再次 `Get` 返回 |
| `ErrPoolTimeout` | 向池提交任务时等待信号量超时 |
| `ErrPoolClosed` | 池已关闭后仍尝试提交 |

## ⚠️ 注意事项

1. **Future 结果只可消费一次**：多次或并发调用 `Get` 时仅有一个调用方能拿到结果，其余返回 `ErrEmpty`。
2. **任务内禁止向同一个池再次提交并阻塞等待其结果**：池的并发由信号量限制，token 由任务自身持有，嵌套提交会因信号量无法释放而死锁。同样**不要在任务内部调用 `pool.Close()`**（会等待自身完成而死锁）。此限制需由调用方自行规避。
3. **panic 恢复**：任务 panic 会被捕获并转为带调用栈的字符串错误返回，原始 panic 值的类型会丢失，无法用 `errors.Is/As` 判断底层错误。
4. **`Submit` 的 ctx 只影响信号量等待**，不会传递给任务本身，也不影响任务的执行过程；任务一旦启动就会执行完毕。
