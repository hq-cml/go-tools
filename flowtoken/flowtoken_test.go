package flowtoken

import (
	"encoding/json"
	"errors"
	"math"
	"sync"
	"testing"
)

// newPrimed 创建一个已完成首次刷新的桶，便于对窗口调整逻辑做确定性测试
// 首次刷新后: cwnd = initTokenNum, tokenNum = cwnd, lastTimeSec = 1
func newPrimed(t *testing.T, initNum, failLimit int64) *FlowTokenBucket {
	t.Helper()
	ftb := NewFlowTokenIdInit("test-host", initNum, failLimit, trigger_failrate)
	if osn := ftb.refreshInitBucket(1); osn == nil {
		t.Fatal("首次刷新应返回非 nil 快照")
	}
	return ftb
}

func TestNewFlowTokenIdDefaults(t *testing.T) {
	ftb := NewFlowTokenId("h")
	if ftb.initTokenNum != init_token_num {
		t.Fatalf("initTokenNum 应为 %d，实际 %d", init_token_num, ftb.initTokenNum)
	}
	if ftb.ssthresh != init_token_num {
		t.Fatalf("ssthresh 应为 %d，实际 %d", init_token_num, ftb.ssthresh)
	}
	if ftb.failLimit != fail_limit {
		t.Fatalf("failLimit 应为 %d，实际 %d", fail_limit, ftb.failLimit)
	}
	if ftb.triggerFailrate != trigger_failrate {
		t.Fatalf("triggerFailrate 应为 %v，实际 %v", trigger_failrate, ftb.triggerFailrate)
	}
	if ftb.cwndMax != default_cwnd_max {
		t.Fatalf("cwndMax 应为 %d，实际 %d", default_cwnd_max, ftb.cwndMax)
	}
}

func TestNewFlowTokenIdInitValidation(t *testing.T) {
	cases := []struct {
		name        string
		initNum     int64
		failLimit   int64
		trigger     float64
		wantInit    int64
		wantFail    int64
		wantTrigger float64
	}{
		{"failLimit 过大", 4000, 300, 0.02, 4000, 199, 0.02},
		{"failLimit 为 0", 4000, 0, 0.02, 4000, fail_limit, 0.02},
		{"failLimit 为负", 4000, -5, 0.02, 4000, fail_limit, 0.02},
		{"trigger 为 0", 4000, 3, 0, 4000, 3, trigger_failrate},
		{"trigger >= 1", 4000, 3, 1.5, 4000, 3, trigger_failrate},
		{"trigger 恰好 1", 4000, 3, 1.0, 4000, 3, trigger_failrate},
		{"trigger 过大", 4000, 3, 0.5, 4000, 3, 0.4},
		{"trigger 恰好 0.4", 4000, 3, 0.4, 4000, 3, 0.4},
		{"trigger 合法", 4000, 3, 0.2, 4000, 3, 0.2},
		{"initNum 超过上限", 20000000, 3, 0.02, default_cwnd_max, 3, 0.02},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ftb := NewFlowTokenIdInit("h", c.initNum, c.failLimit, c.trigger)
			if ftb.initTokenNum != c.wantInit {
				t.Fatalf("initTokenNum 应为 %d，实际 %d", c.wantInit, ftb.initTokenNum)
			}
			if ftb.failLimit != c.wantFail {
				t.Fatalf("failLimit 应为 %d，实际 %d", c.wantFail, ftb.failLimit)
			}
			if ftb.triggerFailrate != c.wantTrigger {
				t.Fatalf("triggerFailrate 应为 %v，实际 %v", c.wantTrigger, ftb.triggerFailrate)
			}
		})
	}
}

func TestGetTriggerRate(t *testing.T) {
	ftb := NewFlowTokenIdInit("h", 4000, 3, 0.2)
	if ftb.GetTriggerFailrate() != 0.2 {
		t.Fatalf("GetTriggerFailrate 应为 0.2，实际 %v", ftb.GetTriggerFailrate())
	}
	if math.Abs(ftb.GetTriggerSuccrate()-0.8) > 1e-9 {
		t.Fatalf("GetTriggerSuccrate 应约为 0.8，实际 %v", ftb.GetTriggerSuccrate())
	}
}

func TestIsTokenExhaustedError(t *testing.T) {
	if _, ok := IsTokenExhaustedError(nil); ok {
		t.Fatal("nil error 不应匹配")
	}
	if _, ok := IsTokenExhaustedError(errors.New("other")); ok {
		t.Fatal("普通 error 不应匹配")
	}
	te := &TokenExhaustedError{Msg: "m"}
	if got, ok := IsTokenExhaustedError(te); !ok || got != te {
		t.Fatal("TokenExhaustedError 应匹配且返回同一对象")
	}
}

func TestTokenExhaustedErrorJSON(t *testing.T) {
	te := &TokenExhaustedError{
		Snapshot: &BucketSnapshot{HostId: "h", Cwnd: 10},
		Msg:      "boom",
	}
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(te.Error()), &m); err != nil {
		t.Fatalf("Error() 应输出合法 JSON，实际: %s, err: %v", te.Error(), err)
	}
	if m["Msg"] != "boom" {
		t.Fatalf("JSON 中 Msg 应为 boom，实际 %v", m["Msg"])
	}
}

func TestGetTokenSuccess(t *testing.T) {
	ftb := NewFlowTokenIdInit("h", 4000, 3, trigger_failrate)
	tok, err := ftb.GetToken()
	if err != nil || tok == nil {
		t.Fatalf("取令牌失败: %v", err)
	}
	if tok.TokenNum != 3999 {
		t.Fatalf("剩余令牌应为 3999，实际: %d", tok.TokenNum)
	}

	tok.Succ()
	if ftb.succCount != 1 {
		t.Fatalf("成功计数应为 1，实际: %d", ftb.succCount)
	}

	tok2, err := ftb.GetToken()
	if err != nil || tok2 == nil {
		t.Fatalf("第二次取令牌失败: %v", err)
	}
	tok2.Fail()
	if ftb.failCount != 1 {
		t.Fatalf("失败计数应为 1，实际: %d", ftb.failCount)
	}
}

// TestGetTokenExhaust 用固定秒调用内部 getToken，避免真实时间跨界导致的不确定性
func TestGetTokenExhaust(t *testing.T) {
	ftb := NewFlowTokenIdInit("h", 3, 3, trigger_failrate)
	for i := 0; i < 3; i++ {
		_, _, _, tok, err := ftb.getToken(100)
		if err != nil {
			t.Fatalf("第 %d 次取令牌不应报错，实际: %v", i+1, err)
		}
		if tok == nil {
			t.Fatalf("第 %d 次取令牌不应为 nil", i+1)
		}
	}

	_, _, _, tok, err := ftb.getToken(100)
	if tok != nil {
		t.Fatal("令牌耗尽时不应返回 token")
	}
	te, ok := IsTokenExhaustedError(err)
	if !ok || te == nil {
		t.Fatalf("应返回 TokenExhaustedError，实际: %v", err)
	}
	if te.Snapshot == nil {
		t.Fatal("错误中应包含快照")
	}
	if ftb.errorCount != 1 {
		t.Fatalf("熔断计数应为 1，实际: %d", ftb.errorCount)
	}
}

// TestGetTokenPublicExhaust 覆盖公开入口 GetToken 的错误分支
func TestGetTokenPublicExhaust(t *testing.T) {
	ftb := NewFlowTokenIdInit("h", 3, 3, trigger_failrate)
	ftb.mu.Lock()
	ftb.lastTimeSec = 1 << 62 // 阻止秒级刷新
	ftb.tokenNum = 0          // 模拟耗尽
	ftb.mu.Unlock()

	_, err := ftb.GetToken()
	if _, ok := IsTokenExhaustedError(err); !ok {
		t.Fatalf("期望 TokenExhaustedError，实际: %v", err)
	}
}

func TestRefreshSameSecond(t *testing.T) {
	ftb := NewFlowTokenIdInit("h", 4000, 3, trigger_failrate)
	if osn := ftb.refreshInitBucket(10); osn == nil {
		t.Fatal("首次刷新应返回非 nil")
	}
	if osn := ftb.refreshInitBucket(10); osn != nil {
		t.Fatal("同一秒内刷新应返回 nil")
	}
	if osn := ftb.refreshInitBucket(9); osn != nil {
		t.Fatal("时间回退不应触发刷新")
	}
}

// 测试"首次刷新"
func TestRefreshFirstSecond(t *testing.T) {
	ftb := NewFlowTokenIdInit("h", 4000, 3, trigger_failrate)
	osn := ftb.refreshInitBucket(100)
	if osn == nil {
		t.Fatal("首次刷新应返回非 nil")
	}
	if ftb.cwnd != 4000 || ftb.tokenNum != 4000 {
		t.Fatalf("首次刷新 cwnd/tokenNum 应为 4000，实际 %d/%d", ftb.cwnd, ftb.tokenNum)
	}
	if ftb.lastTimeSec != 100 {
		t.Fatalf("lastTimeSec 应为 100，实际 %d", ftb.lastTimeSec)
	}
	if osn.LastTimeSec != 0 {
		t.Fatalf("调整前快照的 LastTimeSec 应为 0，实际 %d", osn.LastTimeSec)
	}
}

// 测试全部失败
func TestRefreshAllFail(t *testing.T) {
	ftb := newPrimed(t, 4000, 3)
	ftb.succCount = 0
	ftb.failCount = 4000
	ftb.refreshInitBucket(2)
	if ftb.cwnd != 2000 {
		t.Fatalf("全失败应减半为 2000，实际 %d", ftb.cwnd)
	}
}

func TestRefreshAllFailFloorFailLimit(t *testing.T) {
	ftb := newPrimed(t, 8, 3)

	ftb.succCount = 0
	ftb.failCount = 100
	ftb.refreshInitBucket(2)
	if ftb.cwnd != 4 {
		t.Fatalf("8 减半应为 4，实际 %d", ftb.cwnd)
	}

	ftb.succCount = 0
	ftb.failCount = 100 // 再次减半4->2触发兜底，cwnd=faillimit
	ftb.refreshInitBucket(3)
	if ftb.cwnd != 3 {
		t.Fatalf("4 减半后应不低于 failLimit(3)，实际 %d", ftb.cwnd)
	}
}

func TestRefreshAllSuccGrow(t *testing.T) {
	ftb := newPrimed(t, 4000, 3)
	ftb.succCount = 4000
	ftb.failCount = 0
	ftb.refreshInitBucket(2)
	if ftb.ssthresh != 8000 {
		t.Fatalf("饱和全成功 ssthresh 应翻倍为 8000，实际 %d", ftb.ssthresh)
	}
	if ftb.cwnd != 8000 {
		t.Fatalf("cwnd 应等于 ssthresh(8000)，实际 %d", ftb.cwnd)
	}
}

// 小流量场景，即便成功也不会盲目扩大
func TestRefreshAllSuccNoGrowWhenNotSaturated(t *testing.T) {
	ftb := newPrimed(t, 4000, 3)
	ftb.succCount = 100 // 未达到 ssthresh*0.98
	ftb.failCount = 0
	ftb.refreshInitBucket(2)
	if ftb.ssthresh != 4000 {
		t.Fatalf("未饱和时 ssthresh 应保持不变(4000)，实际 %d", ftb.ssthresh)
	}
	if ftb.cwnd != 4000 {
		t.Fatalf("cwnd 应等于 ssthresh(4000)，实际 %d", ftb.cwnd)
	}
}

// 小流量场景，成功率很大，触发errCnt>0 时 ssthresh 应增长
func TestRefreshAllSuccGrowOnErrCnt(t *testing.T) {
	ftb := newPrimed(t, 4000, 3)
	ftb.succCount = 100
	ftb.failCount = 0
	ftb.errorCount = 500 // 有熔断，说明窗口是瓶颈
	ftb.refreshInitBucket(2)
	if ftb.ssthresh != 4100 {
		t.Fatalf("errCnt>0 时 ssthresh 应增长 100 至 4100，实际 %d", ftb.ssthresh)
	}
	if ftb.cwnd != 4100 {
		t.Fatalf("cwnd 应为 4100，实际 %d", ftb.cwnd)
	}
}

// 增长被 cwndMax 截断
func TestRefreshAllSuccCwndMaxClamp(t *testing.T) {
	ftb := newPrimed(t, 4000, 3)
	ftb.SetCwndMax(5000)
	ftb.succCount = 4000
	ftb.failCount = 0
	ftb.refreshInitBucket(2)
	if ftb.ssthresh != 5000 {
		t.Fatalf("增长应被 cwndMax 截断为 5000，实际 %d", ftb.ssthresh)
	}
	if ftb.cwnd != 5000 {
		t.Fatalf("cwnd 应被截断为 5000，实际 %d", ftb.cwnd)
	}
}

// 部分失败场景，失败数小于成功数，线性减小
func TestRefreshPartFailLinear(t *testing.T) {
	ftb := newPrimed(t, 4000, 3)
	ftb.succCount = 100
	ftb.failCount = 50 // fail < succ，线性减小
	ftb.refreshInitBucket(2)
	if ftb.cwnd != 3950 {
		t.Fatalf("线性减小后 cwnd 应为 3950，实际 %d", ftb.cwnd)
	}
	if ftb.ssthresh != 100 {
		t.Fatalf("ssthresh 应被压低到 succ(100)，实际 %d", ftb.ssthresh)
	}
}

// 部分失败场景，失败数大于等于成功数，指数减小
func TestRefreshPartFailExp(t *testing.T) {
	ftb := newPrimed(t, 4000, 3)
	ftb.succCount = 10
	ftb.failCount = 20 // fail >= succ，指数减小
	ftb.refreshInitBucket(2)
	if ftb.cwnd != 2000 {
		t.Fatalf("指数减小后 cwnd 应为 2000，实际 %d", ftb.cwnd)
	}
}

func TestRefreshSmallSucc(t *testing.T) {
	ftb := newPrimed(t, 3, 3)
	ftb.succCount = 1
	ftb.failCount = 1
	ftb.refreshInitBucket(2)
	if ftb.cwnd != 4 {
		t.Fatalf("小样本有成功时 cwnd 应为 failLimit+succ=4，实际 %d", ftb.cwnd)
	}
}

func TestSetCwndMaxClamp(t *testing.T) {
	ftb := NewFlowTokenIdInit("h", 4000, 3, trigger_failrate)
	ftb.refreshInitBucket(1)

	ftb.SetCwndMax(1000)
	if ftb.ssthresh != 1000 || ftb.cwnd != 1000 || ftb.tokenNum != 1000 {
		t.Fatalf("调小上限应同步 clamp，实际 ssthresh=%d cwnd=%d tokenNum=%d",
			ftb.ssthresh, ftb.cwnd, ftb.tokenNum)
	}

	ftb.SetCwndMax(0) // 0 表示不限制，不应改动现有值
	if ftb.cwnd != 1000 {
		t.Fatalf("设为不限制不应改动现有 cwnd，实际 %d", ftb.cwnd)
	}
}

func TestReportSuccFail(t *testing.T) {
	ftb := NewFlowTokenId("h")
	ftb.ReportSucc()
	ftb.ReportSucc()
	ftb.ReportFail()
	if ftb.succCount != 2 {
		t.Fatalf("成功计数应为 2，实际 %d", ftb.succCount)
	}
	if ftb.failCount != 1 {
		t.Fatalf("失败计数应为 1，实际 %d", ftb.failCount)
	}
}

func TestGetSnapshot(t *testing.T) {
	ftb := NewFlowTokenIdInit("h", 4000, 3, trigger_failrate)
	ftb.refreshInitBucket(1)
	ftb.ReportSucc()
	ftb.ReportFail()

	sn := ftb.GetSnapshot()
	if sn.HostId != "h" {
		t.Fatalf("HostId 应为 h，实际 %s", sn.HostId)
	}
	if sn.Cwnd != 4000 {
		t.Fatalf("Cwnd 应为 4000，实际 %d", sn.Cwnd)
	}
	if sn.SuccCount != 1 || sn.FailCount != 1 {
		t.Fatalf("Succ/Fail 应为 1/1，实际 %d/%d", sn.SuccCount, sn.FailCount)
	}
}

// TestConcurrentAccess 在 -race 下验证并发访问无数据竞争
func TestConcurrentAccess(t *testing.T) {
	ftb := NewFlowTokenIdInit("h", 1000, 3, trigger_failrate)
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				tok, err := ftb.GetToken()
				if err == nil && tok != nil {
					tok.Succ()
				} else {
					ftb.ReportFail()
				}
				ftb.GetSnapshot()
			}
		}()
	}
	wg.Wait()
}
