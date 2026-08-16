# GTX - Goroutine Context

GTX 是一个 Go 语言库，为每个 Goroutine 提供独立的键值存储空间，实现类似 Java ThreadLocal 的功能。

## 特性

- 🚀 轻量级，基于 [concurrent-map](https://github.com/orcaman/concurrent-map) 实现线程安全
- ⚡ 高性能 Goroutine ID 获取（默认汇编直读 `g.goid`，纳秒级）
- 🔒 基于 Goroutine ID 隔离数据，每个 goroutine 拥有独立的存储空间
- 🛡️ 提供安全包装函数 `GoWithGtx`，自动防止内存泄漏
- 📊 支持计数器操作（Incr/Decr）
- 📦 支持任意类型存储（`interface{}`）
- 🔍 支持 JSON 导出当前上下文

## 安装

```bash
go get github.com/hq-cml/go-tools/gtx
```

依赖：
- `github.com/orcaman/concurrent-map` - 线程安全的并发 Map
- `github.com/petermattis/goid` - 高性能 Goroutine ID 获取（汇编直读 `g.goid`）

## 快速开始

```go
func main() {
	// 安全启动 goroutine（推荐）
	gtx.GoWithGtx(func() {
		// 存储数据
		gtx.Set("user_id", 12345)
		gtx.Set("request_id", "abc-123")

		MyFunc()

		// 退出时自动清理，无内存泄漏
	})
	time.Sleep(100 * time.Millisecond)
}

func MyFunc() {
	// 读取数据
	if userID, ok := gtx.Get("user_id"); ok {
		fmt.Printf("User ID: %v\n", userID)
	}
	if reqID, ok := gtx.Get("request_id"); ok {
		fmt.Printf("Req ID: %v\n", reqID)
	}
}
```

## ⚠️ 重要警告：避免内存泄漏

**错误使用会导致严重的内存泄漏！**

### ❌ 错误示例（会导致内存泄漏）

```go
// 危险！数据会永远留在内存中
go func() {
    gtx.Init4Current()  // 初始化
    gtx.Set("key", "value")
    // 忘记调用 Clear4Current()，内存泄漏！
}()
```

### ✅ 正确示例 1：使用安全包装函数（推荐）

```go
// 最简单、最安全的用法
gtx.GoWithGtx(func() {
    gtx.Set("key", "value")
    // 业务逻辑...
    // 函数返回时自动清理
})
```

### ✅ 正确示例 2：手动管理生命周期

```go
go func() {
    defer gtx.Clear4Current()  // 确保退出时清理
    gtx.Init4Current()
    gtx.Set("key", "value")
    // 业务逻辑...
}()
```

## API 文档

### 核心函数

#### `GoWithGtx(fn func())`
安全地启动一个带有 gtx 的 goroutine，自动处理初始化和清理。

```go
gtx.GoWithGtx(func() {
    gtx.Set("data", "value")
    // 自动清理
})
```

#### `GoWithGtxReturn(fn func() interface{}) chan interface{}`
安全地启动 goroutine 并支持返回值。

```go
result := gtx.GoWithGtxReturn(func() interface{} {
    gtx.Set("calc", 1)
    return 42
})
val := <-result  // 42
```

#### `Init4Current()`
为当前 goroutine 初始化上下文。如果已存在则不做任何事。

#### `Clear4Current()`
清理当前 goroutine 的上下文数据，释放内存。

#### `Exist4Current() bool`
检查当前 goroutine 是否已初始化上下文。

### 基础操作

#### `Set(key interface{}, value interface{}) bool`
在当前 goroutine 上下文中存储一个键值对。未初始化时返回 `false`。

```go
gtx.Set("user", "Alice")
```

#### `Get(key interface{}) (interface{}, bool)`
读取当前 goroutine 上下文中的值。未初始化或 key 不存在时返回 `false`。

#### `Del(key interface{}) bool`
删除当前 goroutine 上下文中的 key。

#### `Incr(key interface{}, value int) (int, bool)`
计数器自增，返回自增**之前**的值；key 不存在或类型不匹配时初始化为 `value`。

#### `Decr(key interface{}, value int) (int, bool)`
计数器自减，返回自减**之前**的值；key 不存在或类型不匹配时初始化为 `-value`。

### Goroutine ID

#### `GetGoId() int`
获取当前 goroutine 的 ID。提供两种实现，通过 `DefaultUse` 常量切换：

| DefaultUse | 实现 | 说明 |
|------------|------|------|
| 2（默认） | `github.com/petermattis/goid` | 汇编直读 `g.goid` 字段，纳秒级，推荐 |
| 1 | 解析 `runtime.Stack` | 存在兼容性隐患且性能差，仅作对比/备用 |

### 高级功能

#### `GetCurrCtx() (map[interface{}]interface{}, bool)`
获取当前 goroutine 的完整上下文 map。

#### `JsonCurrent() string`
将当前上下文导出为 JSON 字符串（用于调试）。

```go
gtx.Set("user", "Alice")
gtx.Set("age", 30)
fmt.Println(gtx.JsonCurrent())  // {"user":"Alice","age":30}
```

## 使用场景

### 1. HTTP 请求上下文传递

```go
func main() {
    http.Handle("/", middleware(http.HandlerFunc(handler)))
    http.ListenAndServe(":8080", nil)
}

func middleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        gtx.GoWithGtx(func() {
            gtx.Set("request_id", generateRequestID())
            gtx.Set("user_id", getUserFromToken(r))
            next.ServeHTTP(w, r)
        })
    })
}

func handler(w http.ResponseWriter, r *http.Request) {
    // 随处访问，无需层层传递参数
    if reqID, ok := gtx.Get("request_id"); ok {
        log.Printf("[%v] Processing request", reqID)
    }
}
```

### 2. 链路追踪

```go
func processTask(taskID string) {
    gtx.GoWithGtx(func() {
        gtx.Set("trace_id", generateTraceID())
        gtx.Set("task_id", taskID)
        
        step1()
        step2()
        step3()
    })
}

func step1() {
    if traceID, ok := gtx.Get("trace_id"); ok {
        fmt.Printf("[%v] Step 1 executing\n", traceID)
    }
}
```

### 3. 计数器/统计

无需层层传递计数器参数，调用链中任意深度的函数都能累加，最后统一读取统计结果：

```go
func processOrder() {
    gtx.GoWithGtx(func() {
        fetch()
        parse()
        save()

        if count, ok := gtx.Get("processed_count"); ok {
            fmt.Printf("本次共处理 %v 个步骤\n", count)
        }
    })
}

func fetch() {
    // 多级调用链中累加
    gtx.Incr("processed_count", 1)
}

func parse() {
    gtx.Incr("processed_count", 1)
}

func save() {
    gtx.Incr("processed_count", 1)
}
```

## 注意事项

### 1. 不要跨 Goroutine 共享数据

虽然 `concurrent-map` 是线程安全的，但每个 goroutine 内部的 `map[interface{}]interface{}` 不是。不要将 `GetCurrCtx()` 获取的 map 传给其他 goroutine 使用。

```go
// ❌ 错误：data race
ctx, _ := gtx.GetCurrCtx()
go func() {
    ctx["key"] = "value"  // 危险！
}()
```

### 2. 非持久化存储

gtx 数据仅在单个 goroutine 生命周期内有效，goroutine 结束后数据会被清理（使用 `GoWithGtx` 时）。不能用于：
- 跨请求缓存
- 持久化配置存储
- 全局状态管理

### 3. JSON 序列化限制

`JsonCurrent()` 使用 `encoding/json`，如果存储了不可序列化的值（如 channel、func），会返回空 JSON `{}`。

### 4. 性能考虑

- `GetGoId()` 默认通过汇编直接读取 `g.goid` 字段，纳秒级开销（比 `runtime.Stack` 方案快约 1000 倍）；`runtime.Stack` 方案可通过 `DefaultUse = 1` 切换
- 高频调用场景建议缓存需要的值到局部变量
- 大数据量存储建议使用专门的数据库或缓存服务

## 与标准库 context 的区别

| 特性 | GTX | context.Context |
|------|-----|-----------------|
| 数据传递方式 | 隐式（通过 goroutine ID） | 显式（函数参数传递） |
| 跨函数调用 | 无需修改函数签名 | 需要传递 context 参数 |
| 作用域 | 单个 goroutine | 可以跨 goroutine 传递 |
| 生命周期管理 | 需要手动或自动清理 | 自动（与 context 绑定） |
| 适用场景 | 复杂调用链中的隐式传参 | 请求级别的上下文传递 |

**建议**：
- 新项目优先使用 `context.Context`
- 遗留项目重构或复杂调用链场景考虑使用 GTX

## 示例代码

完整示例见 [demo/main.go](demo/main.go)

```bash
cd demo
go run main.go
```

---

**⚠️ 再次提醒：务必使用 `GoWithGtx` 或手动 `defer Clear4Current()`，否则会导致内存泄漏！**
