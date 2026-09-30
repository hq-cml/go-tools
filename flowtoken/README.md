# flowtoken

基于 TCP 拥塞控制算法的自适应限流库，通过滑动窗口（cwnd）和慢启动阈值（ssthresh）动态调整每秒允许通过的请求数，实现自动熔断与恢复。

---

## 目录

- [快速开始](#快速开始)
- [核心概念](#核心概念)
- [窗口调整算法](#窗口调整算法)
- [完整生命周期示例](#完整生命周期示例)
- [API 参考](#api-参考)
- [Manager 管理器](#manager-管理器)
- [已知问题与注意事项](#已知问题与注意事项)

---

## 快速开始

### 安装

```bash
go get github.com/hq-cml/go-tools
```

### 基本用法

```go
package main

import "github.com/hq-cml/go-tools/flowtoken"

func main() {
    // 创建 FlowToken 实例
    ft := flowtoken.NewFlowTokenId("my-service")

    // 获取令牌
    t, err := ft.GetToken()
    if err != nil {
        // 令牌耗尽，触发熔断，请求被拒绝
        return err
    }

    // 必须在请求结束后上报结果
    defer func() {
        if err == nil {
            t.Succ() // 请求成功
        } else {
            t.Fail() // 请求失败
        }
    }()

    // 执行实际业务请求...
    err = doRequest()
    return err
}
```

### 基于 Manager（推荐）

实际业务中通常有多个下游实例，用 `FtkManager` 按 host 维度统一管理是最常用的方式：

```go
package main

import "github.com/hq-cml/go-tools/flowtoken"

// 全局共享的流控管理器，按下游实例（host）维度管理多个流控桶
var manager = newFlowTokenManager()

func newFlowTokenManager() *flowtoken.FtkManager {
    m := &flowtoken.FtkManager{
        InitTokenNum:  4000, // 每个下游实例的初始窗口
        InitFailLimit: 3,    // 小样本失败阈值
    }
    m.Start() // 归一化参数（无效值回退默认），也可省略
    return m
}

// doRequest 对指定下游实例发起一次受流控保护的请求
func doRequest(host string) error {
    // 获取该下游的流控桶（幂等，不存在则自动创建）
    ft := manager.AddOrInitDefault(host)

    // 获取令牌
    t, err := ft.GetToken()
    if err != nil {
        // 令牌耗尽，触发熔断，请求被拒绝
        return err
    }

    // 请求结束后必须上报结果
    defer func() {
        if err == nil {
            t.Succ() // 成功，窗口倾向于放大
        } else {
            t.Fail() // 失败，窗口倾向于收缩
        }
    }()

    // 执行实际业务请求...
    return callDownstream(host)
}
```

> 每个 host 独立限流，互不影响。多个下游实例同时过载时，各自按自己的健康状态收敛，不会互相拖累。

### 运行测试

```bash
go test -cover -count=1 ./...
```

---

## 核心概念

FlowToken 借鉴 TCP 拥塞控制算法，核心是三个变量的动态调整：

| 变量 | 含义 | 通俗理解 |
|------|------|----------|
| `cwnd` | 拥塞窗口 | **这一秒能放多少请求**（每秒允许的请求数上限） |
| `ssthresh` | 慢启动阈值 | **历史最高水位线**（记录曾经健康承受过的压力） |
| `tokenNum` | 剩余令牌 | 每秒初重置为 `cwnd`，每次 `GetToken` 减 1，耗尽则熔断 |
| `initTokenNum` | 初始令牌数 | 初始窗口大小和 `ssthresh` 初始值（默认 4000） |
| `failLimit` | 失败阈值 | 小样本判断的基准值（默认 3） |
| `triggerFailrate` | 触发失败率 | 区分"全成功/部分失败/全失败"的阈值（默认 0.02） |
| `cwndMax` | 窗口上限 | `cwnd`/`ssthresh` 的最大值（默认 10000000，0 表示不限制） |

### 每秒决策机制

```
第0秒:  启动，cwnd=initTokenNum, ssthresh=initTokenNum
第1秒:  统计上一秒的 succ/fail → 调整 cwnd 和 ssthresh → tokenNum=cwnd
        放请求，每个请求 tokenNum--
第2秒:  统计上一秒的 succ/fail → 调整 cwnd 和 ssthresh → tokenNum=cwnd
        ...
```

每秒开头由 `refreshInitBucket()` 执行窗口调整，核心思想是 **AIMD（Additive Increase, Multiplicative Decrease）**：增长慢、减小快，保证稳定性。

---

## 窗口调整算法

每秒根据上一秒的 `succ`（成功数）和 `fail`（失败数）统计，走以下几个分支之一：

### 1. 刚启动（lastTimeSec == 0）

```go
t.cwnd = t.initTokenNum  // 直接使用初始值
```

首次请求时直接使用初始值。`ssthresh` 在构造函数里已设为 `initTokenNum`，所以初始状态下 `cwnd = ssthresh = initTokenNum = 4000`。

### 2. 全失败 allFail → 服务疑似宕机

**触发条件**：失败率 >= 98%（`GetTriggerSuccrate = 1 - triggerFailrate`），或小样本时 `fail >= failLimit 且 succ < failLimit`

```go
t.cwnd >>= 1              // 指数减半
if t.cwnd < t.failLimit {
    t.cwnd = t.failLimit  // 最低保 3
}
// ssthresh 不变，记住历史最高水位，为恢复留目标
```

特点：快速收敛熔断，每秒减半，类似 TCP 的"乘性减小"。

**数字举例**：服务持续宕机，每秒所有请求全部失败，`initTokenNum=4000`：

| 秒 | succ | fail | 动作 | cwnd | ssthresh |
|----|------|------|------|------|----------|
| 1 | - | - | 启动 | 4000 | 4000 |
| 2 | 0 | 4000 | 全失败，减半 | 2000 | 4000 |
| 3 | 0 | 2000 | 全失败，减半 | 1000 | 4000 |
| 4 | 0 | 1000 | 全失败，减半 | 500 | 4000 |
| 5 | 0 | 500 | 全失败，减半 | 250 | 4000 |
| 6 | 0 | 250 | 全失败，减半 | 125 | 4000 |
| 7 | 0 | 125 | 全失败，减半 | 62 | 4000 |
| ... | ... | ... | ... | ... | ... |
| N | 0 | 3 | 全失败，减半到保底 | 3 | 4000 |

可以看到 `cwnd` 从 4000 逐秒减半：4000 → 2000 → 1000 → 500 → 250 → 125 → 62 → ... → 3，最终稳定在 `failLimit=3`。而 `ssthresh` 始终保持 4000 不变，为服务恢复时提供快速跳回的目标。

### 3. 全成功 allSucc → 服务健康，快速启动

**触发条件**：失败率 <= 2%（`triggerFailrate`），或小样本时 `fail < failLimit 且 succ >= failLimit`

```go
if succ >= int64(float64(t.ssthresh)*t.GetTriggerSuccrate()) || errCnt > 0 {
    t.ssthresh += succ     // ssthresh 指数增长
    if t.cwndMax > 0 && t.ssthresh > t.cwndMax {
        t.ssthresh = t.cwndMax  // 超过上限则截断
    }
}
t.cwnd = t.ssthresh          // cwnd 直接拉到 ssthresh
```

特点：只要不是全失败，窗口立即跳回历史水位（快速恢复）；若成功数超过水位线，水位线持续增长。

全成功有两种典型场景，效果不同：

**场景 A：从熔断中快速恢复**（succ 达不到 ssthresh*0.98，不增长 ssthresh）

假设服务从宕机恢复，`cwnd` 刚降到 125，`ssthresh=4000`，本秒 125 个请求全部成功：

| 秒 | succ | fail | 判断 | ssthresh 增长？ | cwnd | ssthresh |
|----|------|------|------|-----------------|------|----------|
| 6 | 0 | 125 | 全失败 | - | 125 | 4000 |
| 7 | 125 | 0 | 全成功 | 125 < 4000*0.98=3920，**不增长** | **4000** | 4000 |
| 8 | 4000 | 0 | 全成功 | 4000 >= 3920，**增长** | 8000 | 8000 |

关键：第 7 秒 `succ=125` 虽然达不到 `3920`，但只要判定为"全成功"，`cwnd` 就直接跳到 `ssthresh=4000`（快速恢复），只是 `ssthresh` 不增长。第 8 秒 `succ=4000` 达到水位线，`ssthresh` 才开始增长。

**场景 B：服务能力持续提升**（succ 超过 ssthresh*0.98，ssthresh 指数增长）

| 秒 | succ | fail | 动作 | cwnd | ssthresh |
|----|------|------|------|------|----------|
| 1 | - | - | 启动 | 4000 | 4000 |
| 2 | 4000 | 0 | succ≥3920，ssthresh+=4000 | 8000 | 8000 |
| 3 | 8000 | 0 | succ≥7840，ssthresh+=8000 | 16000 | 16000 |
| 4 | 16000 | 0 | succ≥15680，ssthresh+=16000 | 32000 | 32000 |

`ssthresh += succ`，每秒翻倍增长，类似 TCP 慢启动。`cwnd` 跟着 `ssthresh` 走，窗口快速放大。

### 4. 部分失败 partFail → 压力临界，线性微调

**触发条件**：不是全失败，也不是全成功，失败率 > 2%

```go
if fail >= succ {
    t.cwnd >>= 1              // 失败占多数，指数减半
} else {
    t.cwnd -= fail            // 失败少，线性减小
    if t.ssthresh > succ {
        t.ssthresh = succ     // 下调 ssthresh
    }
}
if t.cwnd < t.failLimit {
    t.cwnd = t.failLimit      // 最低保 3
}
```

特点：线性减小（`cwnd -= fail`），比指数减半温和，适合"网络抖动"而非"宕机"。同时下调 `ssthresh`，降低未来的恢复目标。

`partFail` 内部又分两种情况：

**情况 A：失败占多数（fail >= succ）→ 指数减半**

| 秒 | succ | fail | 动作 | cwnd | ssthresh |
|----|------|------|------|------|----------|
| 1 | - | - | 启动 | 4000 | 4000 |
| 2 | 1000 | 1500 | fail>succ，cwnd减半 | 2000 | 4000 |
| 3 | 500 | 800 | fail>succ，cwnd减半 | 1000 | 4000 |

`fail >= succ` 时走 `cwnd >>= 1`，和全失败一样的指数减半，但 `ssthresh` 不变。

**情况 B：失败少成功多（fail < succ）→ 线性减小 + 下调 ssthresh**

| 秒 | succ | fail | 动作 | cwnd | ssthresh |
|----|------|------|------|------|----------|
| 1 | - | - | 启动 | 4000 | 4000 |
| 2 | 3800 | 200 | fail<succ，cwnd-=200 | 3800 | min(4000,3800)=3800 |
| 3 | 3600 | 200 | fail<succ，cwnd-=200 | 3600 | min(3800,3600)=3600 |
| 4 | 3400 | 200 | fail<succ，cwnd-=200 | 3400 | min(3600,3400)=3400 |

关键区别：`cwnd -= fail`（减去失败数，而非减半），每秒只减 200，比指数减半温和得多。同时 `ssthresh` 也跟着下调到 `succ`，降低恢复目标——表示"这个服务现在只能扛 3800 了，不再是 4000"。

### 5. 偶尔成功（succ > 0 但不满足以上任何条件）

```go
if t.cwnd <= t.failLimit {
    t.cwnd = t.failLimit + succ   // 留一点试探空间
}
// else cwnd 不小，不做调整
```

请求量极少（每秒就 1-2 个），不好判断服务状态，给一点点窗口试探。

**数字举例**：

| 秒 | succ | fail | cwnd 调整前 | 判断 | 动作 | cwnd 调整后 |
|----|------|------|------------|------|------|------------|
| 6 | 1 | 1 | 3 | 非 allFail、非 allSucc、非 partFail，succ>0 | cwnd=3+1 | 4 |
| 7 | 2 | 0 | 4 | allSucc?（succ<3 否），succ>0 | cwnd>3，不动 | 4 |

窗口在 3-4 附近试探性波动，等待足够样本后再做判断。

### 决策流程图

```
                    每秒统计 succ/fail
                          │
        ┌─────────────────┼──────────────────┐
        ▼                 ▼                  ▼
     全失败             部分失败            全成功
  failRate>=98%     2%<failRate<98%     failRate<=2%
        │                 │                  │
   cwnd >>= 1      fail>succ: cwnd>>=1    ssthresh += succ
   (指数减半)      fail<succ: cwnd-=fail  cwnd = ssthresh
                  (线性减小)           (指数增长/快速恢复)
        │                 │                  │
        └─────────────────┼──────────────────┘
                          ▼
                  tokenNum = cwnd
                  (开启新的一秒放行)
```

---

## 完整生命周期示例

假设服务正常 → 宕机 → 恢复，`initTokenNum=4000`：

| 秒 | 上一秒 succ | 上一秒 fail | 判断 | 本秒 cwnd | 本秒 ssthresh | 说明 |
|----|-------------|-------------|------|-----------|---------------|------|
| 1 | - | - | 启动 | 4000 | 4000 | 初始 |
| 2 | 4000 | 0 | 全成功 | 8000 | 8000 | succ≥3920，ssthresh 增长 |
| 3 | 8000 | 0 | 全成功 | 16000 | 16000 | 继续增长 |
| 4 | 0 | 16000 | 全失败 | 8000 | 16000 | cwnd 减半，ssthresh 不变 |
| 5 | 0 | 8000 | 全失败 | 4000 | 16000 | cwnd 减半 |
| 6 | 0 | 4000 | 全失败 | 2000 | 16000 | cwnd 减半 |
| 7 | 1000 | 1000 | 部分失败 | 1000 | 16000 | fail≥succ，cwnd 减半 |
| 8 | 1000 | 0 | 全成功 | 16000 | 16000 | succ<15680，cwnd 快速恢复到 ssthresh |
| 9 | 16000 | 0 | 全成功 | 32000 | 32000 | succ≥15680，ssthresh 增长 |

可以看到：**宕机时快速减半（16000→8000→4000→2000），恢复时 cwnd 快速跳回 ssthresh（1000→16000），随后指数增长（16000→32000）**。

---

## 已知问题与注意事项

### 1. GetTriggerSuccrate 命名易误解

```go
func (ftb *FlowTokenBucket) GetTriggerSuccrate() float64 {
    return 1 - ftb.triggerFailrate  // triggerFailrate=0.02 → 返回 0.98
}
```

此值同时用作 `allFail` 判断的失败率上限阈值（失败率 >= 98% 才算全失败），命名和语义容易误解。

### 2. 魔法数字 200

`allFail`/`allSucc`/`partFail` 硬编码 `200` 作为小样本/大样本分界线，与 `failLimit=3` 耦合。当 `failLimit` 被调大时（上限 199），小样本分支的语义会发生变化。
