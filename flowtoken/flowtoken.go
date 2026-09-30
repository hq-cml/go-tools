package flowtoken

import (
	"encoding/json"
	log "github.com/hq-cml/go-tools/logger"
	"sync"
	"time"
)

const (
	init_token_num   = 4000     // 默认初始令牌数，即初始滑动窗口大小
	fail_limit       = 3        // 默认失败阈值，小样本时失败数达到此值即触发窗口调整
	trigger_failrate = 0.02     // 默认触发失败率阈值，用于划分三种窗口调整策略
	default_cwnd_max = 10000000 // 默认最大窗口大小
	log_format       = "[FlowTokenBucket] %s. hostId=%s, currSec=%d, last_cwnd=%d, cwnd=%d, ssthresh=%d, request=%d, succ=%d, fail=%d, error=%d"
)

// IsTokenExhaustedError 判断 error 是否为 TokenExhaustedError 类型
// 返回转换后的错误对象和是否匹配的布尔值
func IsTokenExhaustedError(e error) (*TokenExhaustedError, bool) {
	if e == nil {
		return nil, false
	}
	te, ok := e.(*TokenExhaustedError)
	return te, ok
}

// TokenExhaustedError 令牌耗尽错误
// 当令牌用完（限流触发）或低优先级请求被丢弃时返回此错误
type TokenExhaustedError struct {
	Snapshot *BucketSnapshot // 触发限流时的当前快照
	Msg      string          // 错误描述信息
}

// Error 实现error接口，将错误信息序列化为JSON格式输出
func (e *TokenExhaustedError) Error() string {
	s := ""
	sb, _ := json.Marshal(e)
	if sb != nil {
		s = string(sb)
	}
	return s
}

// Token 表示获取到的一个令牌
// 调用方拿到 Token 后需在请求结束时调用 Succ() 或 Fail() 上报结果
type Token struct {
	TokenNum  int64            // 获取令牌后剩余的令牌数
	flowToken *FlowTokenBucket // 关联的 FlowTokenBucket 实例，用于结果上报
}

/*
 *
 * triggerFailrate 触发失败率阈值，用于划分三种窗口调整策略：
 * 以 triggerFailrate=0.2 为例（GetTriggerSuccrate = 1 - 0.2 = 0.8）：
 * 1. all fail:  failRate >= 0.8 时，指数减小窗口（下游实例疑似宕机）
 * 2. all succ:  failRate <= 0.2 时，指数增加窗口（下游健康，可承受更大压力）
 * 3. part fail: 0.2 < failRate < 0.8 时，线性减小窗口（压力临界，小幅度回调）
 */
// FlowTokenBucket 流控令牌桶，核心借鉴 TCP 拥塞控制算法
// 通过滑动窗口(cwnd)和慢启动阈值(ssthresh)动态调整每秒允许通过的请求数
// 秒级别令牌桶，用于控制每秒的请求数，并且每秒都会根据上一秒的情况动态刷新
type FlowTokenBucket struct {
	hostId          string      // 客户端标识，用于日志和PC上报，本质上他标识一个下游实例（比如：一个ip:port，一个url等）
	mu              *sync.Mutex // 互斥锁，保护所有字段的并发访问
	lastTimeSec     int64       // 上次请求的秒级时间戳，用于检测秒级切换刷新
	tokenNum        int64       // 当前秒：剩余令牌数，每秒初重置为cwnd，每次GetToken减1
	cwnd            int64       // 拥塞窗口大小，当前秒允许的请求数上限
	ssthresh        int64       // 慢启动阈值，曾经成功率高的时候理想预制，用于恢复后的窗口恢复
	requestCount    int64       // 当前秒：请求次数（包括令牌耗尽的情况）
	succCount       int64       // 当前秒：成功次数
	failCount       int64       // 当前秒：失败次数
	errorCount      int64       // 当前秒：熔断次数（令牌耗尽的次数）
	cwndMax         int64       // 令牌数上限，0表示不限制
	initTokenNum    int64       // 初始令牌数，用于初始化和恢复
	failLimit       int64       // 失败阈值，小样本判断的基准
	triggerFailrate float64     // 触发失败率阈值，默认0.02
}

// BucketSnapshot 流控状态快照，记录某一时刻的流控指标
type BucketSnapshot struct {
	HostId       string
	LastTimeSec  int64 // 上次请求时间戳
	TokenNum     int64 // 令牌控制（当前剩余令牌数）
	Cwnd         int64 // 拥塞窗口
	Ssthresh     int64 // 慢启动阈值
	RequestCount int64 // 请求次数（包括令牌耗尽情况）
	SuccCount    int64 // 成功次数
	FailCount    int64 // 失败次数
	ErrorCount   int64 // 熔断次数
}

// NewFlowTokenId 创建普通模式的FlowToken，使用默认参数
func NewFlowTokenId(hostId string) *FlowTokenBucket {
	return NewFlowTokenIdInit(hostId, int64(init_token_num), int64(fail_limit), trigger_failrate)
}

// NewFlowTokenIdInit 创建普通模式的FlowToken，可自定义初始令牌数和失败阈值
// initTokenNum: 初始令牌数（即初始窗口大小和ssthresh初始值）
// failLimit: 失败阈值，会被限制在 [1, 199] 范围内
func NewFlowTokenIdInit(hostId string, initTokenNum, failLimit int64, triggerFailrate float64) *FlowTokenBucket {
	if failLimit >= 200 {
		failLimit = 199
	}
	if failLimit <= 0 {
		failLimit = fail_limit
	}
	if triggerFailrate <= 0 || triggerFailrate >= 1 {
		triggerFailrate = trigger_failrate
	}
	if triggerFailrate > 0.4 {
		triggerFailrate = 0.4
	}
	if initTokenNum > default_cwnd_max {
		initTokenNum = default_cwnd_max
	}

	t := &FlowTokenBucket{
		hostId: hostId,
		mu:     &sync.Mutex{},
		//cwnd:     initTokenNum,  // cwnd在initToken首次调用时设置
		ssthresh:        initTokenNum, // ssthresh初始值为initTokenNum
		initTokenNum:    initTokenNum,
		failLimit:       failLimit,
		triggerFailrate: triggerFailrate, // 默认触发失败率2%
		cwndMax:         default_cwnd_max,
	}
	log.Info("create flowtoken bucket %s", hostId)
	return t
}

// GetTriggerSuccrate 返回触发"全成功"判断的成功率阈值 = 1 - triggerFailrate
// 注意：此值也用作allFail判断的失败率上限阈值
// 例如 triggerFailrate=0.02 时，返回0.98，失败率>=98%才认为全失败
func (ftb *FlowTokenBucket) GetTriggerSuccrate() float64 {
	return 1 - ftb.triggerFailrate
}

// GetTriggerFailrate 返回触发失败率阈值
func (ftb *FlowTokenBucket) GetTriggerFailrate() float64 {
	return ftb.triggerFailrate
}

// allFail 判断普通模式下是否为"全失败"（服务疑似宕机）
func (ftb *FlowTokenBucket) allFail(succ, fail int64) bool {
	total := succ + fail
	if total == 0 {
		return false
	}

	if total < 200 {
		failRate := float64(fail) / float64(total)
		if failRate >= ftb.GetTriggerSuccrate() {
			return true
		}

		return fail >= ftb.failLimit && succ < ftb.failLimit
	} else {
		failRate := float64(fail) / float64(total)
		if failRate >= ftb.GetTriggerSuccrate() {
			return true
		} else {
			return false
		}
	}
}

// allSucc 判断是否为"全成功"（服务健康，可承受更大压力）
// - 小样本(total<200): fail < failLimit 且 succ >= failLimit
// - 大样本(total>=200): 失败率 <= triggerFailrate(默认0.02)
func (ftb *FlowTokenBucket) allSucc(succ, fail int64) bool {
	total := succ + fail
	if total < 200 {
		return fail < ftb.failLimit && succ >= ftb.failLimit
	} else {
		failRate := float64(fail) / float64(total)
		if failRate <= ftb.GetTriggerFailrate() {
			return true
		} else {
			return false
		}
	}
}

// partFail 判断是否为"部分失败"（有成功也有失败，且失败较多，处于压力临界点）
// 如果请求量小于每秒钟200个，则每秒失败3个就开始降低阈值
// 否则按百分比计算，2%及以上的请求失败则开始降低阈值
func (ftb *FlowTokenBucket) partFail(succ, fail int64) bool {
	total := succ + fail
	if total < 200 {
		return fail >= ftb.failLimit
	} else {
		failRate := float64(fail) / float64(total)
		if failRate <= ftb.GetTriggerFailrate() {
			return false
		} else {
			return true
		}
	}
}

// refreshInitBucket 普通模式秒级窗口调整
// 这是核心的拥塞控制逻辑，借鉴TCP拥塞控制的三种策略：
// 1. 全失败 → 指数减半（快速收敛）
// 2. 全成功 → 指数增长（快速启动）
// 3. 部分失败 → 线性减小（压力临界微调）
// 整个函数有外层锁保护
func (ftb *FlowTokenBucket) refreshInitBucket(currSec int64) *BucketSnapshot {
	// 相同的秒内，直接返回
	if currSec <= ftb.lastTimeSec {
		return nil
	}

	// 新的一秒开始，启动刷新
	// 保存调整前的快照
	osn := &BucketSnapshot{
		HostId:       ftb.hostId,
		LastTimeSec:  ftb.lastTimeSec,
		TokenNum:     ftb.tokenNum,
		Cwnd:         ftb.cwnd,
		Ssthresh:     ftb.ssthresh,
		RequestCount: ftb.requestCount,
		SuccCount:    ftb.succCount,
		FailCount:    ftb.failCount,
		ErrorCount:   ftb.errorCount,
	}

	// 取出上一秒的统计数据，然后清零
	succ := ftb.succCount
	fail := ftb.failCount
	errCnt := ftb.errorCount
	ftb.requestCount = 0
	ftb.succCount = 0
	ftb.failCount = 0
	ftb.errorCount = 0
	if ftb.lastTimeSec == 0 {
		// 刚启动，窗口设为初始值
		ftb.cwnd = ftb.initTokenNum
	} else if ftb.allFail(succ, fail) {
		// 全失败，服务宕机，指数衰减，减半窗口
		ftb.cwnd >>= 1
		if ftb.cwnd < ftb.failLimit {
			ftb.cwnd = ftb.failLimit
		}
	} else if ftb.allSucc(succ, fail) {
		// 全成功(偶尔失败也算)，快速启动算法
		// 为什么需要succ>=0.98*ssthresh：
		//   因为全成功时只有成功数达到 ssthresh 的 98%（或上秒有熔断）才翻倍增长，
		//   成功数太少则维持原值，避免在请求量不足时盲目放大窗口。
		// 为什么需要errCnt>0也可以：
		//   errCnt>0 意味着上一秒有请求因为令牌耗尽被拒绝，即"需求超过了窗口"，窗口本身就是瓶颈。
		//   而此刻又是 allSucc（下游健康），说明拒绝不是下游的问题，而是我们放行得太少，所以也需要放大。
		if succ >= int64(float64(ftb.ssthresh)*ftb.GetTriggerSuccrate()) || errCnt > 0 {
			// 成功数超过承受压力指数，表示可承受压力上涨，指数级别放大ssthresh

			ftb.ssthresh += succ
			if ftb.cwndMax > 0 && ftb.ssthresh > ftb.cwndMax {
				ftb.ssthresh = ftb.cwndMax
			}
		}
		ftb.cwnd = ftb.ssthresh
	} else if ftb.partFail(succ, fail) {
		// 部分失败：有成功也有失败请求，并且失败数较多，达到服务或网络可承受压力临界点
		// 按失败比率减小
		if fail >= succ {
			// 失败占多数，指数减小滑动窗口
			ftb.cwnd >>= 1
		} else {
			// 小幅度波动，线性减小活动窗口，此时承受压力具有参考价值
			ftb.cwnd -= fail
			if ftb.ssthresh > succ {
				ftb.ssthresh = succ
			}
		}

		// 不是宕机，而是抖动，一次太少了，保证每秒有3次试探式的请求控制
		// 下次可能偶尔成功，可能全失败
		if ftb.cwnd < ftb.failLimit {
			ftb.cwnd = ftb.failLimit
		}
	} else if succ > 0 {
		// 偶尔成功，请求量太少了，但成功了1,2次
		// 如果cwnd也很小，但只要有成功，就设置临界点，期望下次能用上全成功，或者网络抖动又回来了
		if ftb.cwnd <= ftb.failLimit {
			ftb.cwnd = ftb.failLimit + succ
		}
		// else 如果cwnd也不小，要么超时居然没反馈，要么都被熔断了，先不管了
	}
	// else 没有成功，且fail=1,2次，属于窗口较小情况，或者请求本来就少

	// 应用令牌数上限
	ftb.tokenNum = ftb.cwnd
	ftb.lastTimeSec = currSec

	return osn
}

// GetToken 获取一个令牌，是外部调用的主入口
// 返回 Token 和 error，当令牌耗尽或被优先级丢弃时返回 TokenExhaustedError
// 调用方应在请求结束后调用 Token.Succ() 或 Token.Fail() 上报结果
func (ftb *FlowTokenBucket) GetToken() (*Token, error) {
	currSec := time.Now().Unix()
	faillimit, osn, nsn, ft, err := ftb.getToken(currSec)
	if err != nil {
		// 是否需要日志？外层打印更合适
		return nil, err
	}

	// osn != nil 表示发生了秒级切换，需要输出日志和上报PC
	if osn != nil {
		// 将日志打印移出锁，减少锁持有时间
		if osn.Cwnd <= faillimit && nsn.Cwnd > faillimit {
			// 窗口从低位恢复到正常，记录启动日志
			log.Info(log_format, "startup", nsn.HostId, currSec, osn.Cwnd, nsn.Cwnd, nsn.Ssthresh, osn.RequestCount, osn.SuccCount, osn.FailCount, osn.ErrorCount)
		} else if nsn.Cwnd <= faillimit && osn.ErrorCount > 0 {
			// 窗口降到最低且有熔断，记录熔断日志
			log.Warn(log_format, "shutdown", nsn.HostId, currSec, osn.Cwnd, nsn.Cwnd, nsn.Ssthresh, osn.RequestCount, osn.SuccCount, osn.FailCount, osn.ErrorCount)
		} else {
			// 正常运行日志
			log.Info(log_format, "running", nsn.HostId, currSec, osn.Cwnd, nsn.Cwnd, nsn.Ssthresh, osn.RequestCount, osn.SuccCount, osn.FailCount, osn.ErrorCount)
		}
	}
	return ft, err
}

// getToken 获取令牌的内部实现（加锁版）
// 返回值: failLimit, 调整前快照, 调整后快照, Token, error
// 调用方为 GetToken，此方法在锁内执行
func (ftb *FlowTokenBucket) getToken(currSec int64) (int64, *BucketSnapshot, *BucketSnapshot, *Token, error) {
	ftb.mu.Lock()
	defer ftb.mu.Unlock()
	// 秒级切换时调整窗口，返回调整前快照
	oldSnapshot := ftb.refreshInitBucket(currSec)
	ftb.requestCount++
	if ftb.tokenNum <= 0 {
		// 令牌耗尽，触发熔断
		ftb.errorCount++
		sn := &BucketSnapshot{
			HostId:       ftb.hostId,
			LastTimeSec:  ftb.lastTimeSec,
			TokenNum:     ftb.tokenNum,
			Cwnd:         ftb.cwnd,
			Ssthresh:     ftb.ssthresh,
			RequestCount: ftb.requestCount,
			SuccCount:    ftb.succCount,
			FailCount:    ftb.failCount,
			ErrorCount:   ftb.errorCount,
		}
		return 0, nil, nil, nil, &TokenExhaustedError{
			Snapshot: sn,
			Msg:      ftb.hostId + ":token exhausted",
		}
	}

	// 消耗一个令牌
	ftb.tokenNum--
	token := &Token{
		TokenNum:  ftb.tokenNum,
		flowToken: ftb,
	}

	// 如果发生了秒级切换，构建调整后快照用于日志输出
	var newSnapshot *BucketSnapshot
	if oldSnapshot != nil {
		newSnapshot = &BucketSnapshot{
			HostId:       ftb.hostId,
			LastTimeSec:  ftb.lastTimeSec,
			TokenNum:     ftb.tokenNum,
			Cwnd:         ftb.cwnd,
			Ssthresh:     ftb.ssthresh,
			RequestCount: ftb.requestCount,
			SuccCount:    ftb.succCount,
			FailCount:    ftb.failCount,
			ErrorCount:   ftb.errorCount,
		}
	}

	return ftb.failLimit, oldSnapshot, newSnapshot, token, nil
}

// ReportSucc 上报请求成功（外部接口）
func (ftb *FlowTokenBucket) ReportSucc() {
	ftb.reportSucc()
}

// reportSucc 上报成功的内部实现（加锁）
// succCount++，如果 passAsManyAsPossible 为 true 则 tokenNum++ 退还令牌
// tokenNum++倾向于尽可能地放过更多的请求
// 去掉此行，flowtoken会严格地按tokenNum过滤每秒的请求
func (ftb *FlowTokenBucket) reportSucc() {
	ftb.mu.Lock()
	defer ftb.mu.Unlock()
	ftb.succCount++
}

// ReportFail 上报请求失败（外部接口）
func (ftb *FlowTokenBucket) ReportFail() {
	ftb.reportFail()
}

// reportFail 上报失败的内部实现（加锁）
func (ftb *FlowTokenBucket) reportFail() {
	ftb.mu.Lock()
	defer ftb.mu.Unlock()
	ftb.failCount++
}

// Succ Token的方法，上报请求成功，委托给关联的 FlowTokenBucket
func (t *Token) Succ() {
	t.flowToken.reportSucc()
}

// Fail Token的方法，上报请求失败，委托给关联的 FlowTokenBucket
func (t *Token) Fail() {
	t.flowToken.reportFail()
}

// SetCwndMax 设置窗口上限，防止窗口过大，0表示不限制
func (ftb *FlowTokenBucket) SetCwndMax(num int64) {
	ftb.mu.Lock()
	defer ftb.mu.Unlock()
	ftb.cwndMax = num
	if num > 0 {
		if ftb.ssthresh > num {
			ftb.ssthresh = num
		}
		if ftb.cwnd > num {
			ftb.cwnd = num
		}
		if ftb.tokenNum > num {
			ftb.tokenNum = num
		}
	}
}

// GetSnapshot 获取当前流控状态的快照
// 这个方法一般是用户手工触发的（想看一下滑动窗口当前的状态），不会频繁调用
// 所以暂时不加读写锁了，直接加锁
// TODO read write lock?
func (ftb *FlowTokenBucket) GetSnapshot() *BucketSnapshot {
	ftb.mu.Lock()
	defer ftb.mu.Unlock()
	return &BucketSnapshot{
		HostId:       ftb.hostId,
		LastTimeSec:  ftb.lastTimeSec,
		TokenNum:     ftb.tokenNum,
		Cwnd:         ftb.cwnd,
		Ssthresh:     ftb.ssthresh,
		RequestCount: ftb.requestCount,
		SuccCount:    ftb.succCount,
		FailCount:    ftb.failCount,
		ErrorCount:   ftb.errorCount,
	}
}
