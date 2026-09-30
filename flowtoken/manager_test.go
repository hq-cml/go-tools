package flowtoken

import (
	"fmt"
	"testing"
)

// TestManagerMultiHostFlowControl 演示：模拟 10 个下游实例，用 Manager 按 host 维度做独立流控。
// 同一秒内每个实例最多放行 initToken 个请求，其余被令牌耗尽拒绝。
func TestManagerMultiHostFlowControl(t *testing.T) {
	const (
		hostCount  = 10
		initToken  = int64(100)
		reqPerHost = 300
	)

	m := &FtkManager{}
	if err := m.Start(); err != nil {
		t.Fatalf("Start 失败: %v", err)
	}

	// 为每个下游实例建一个独立的流控桶
	for i := 0; i < hostCount; i++ {
		host := fmt.Sprintf("downstream-%d", i)
		if ft := m.AddOrInit(host, initToken, fail_limit, trigger_failrate); ft == nil {
			t.Fatalf("host %s 应返回非 nil 桶", host)
		}
	}

	// 幂等：用不同参数重复 AddOrInit，应返回已存在的实例，参数被忽略
	ft := m.AddOrInit("downstream-0", 9999, 3, 0.02)
	if ft.initTokenNum != initToken {
		t.Fatalf("重复 AddOrInit 不应重建，initTokenNum 应保持 %d，实际 %d", initToken, ft.initTokenNum)
	}

	// 各实例独立限流：同一秒内每个 host 最多放行 initToken 个请求
	for i := 0; i < hostCount; i++ {
		host := fmt.Sprintf("downstream-%d", i)
		ft := m.Get(host)
		passed, rejected := 0, 0
		for j := 0; j < reqPerHost; j++ {
			tok, err := ft.GetToken()
			if err != nil {
				if _, ok := IsTokenExhaustedError(err); !ok {
					t.Fatalf("host %s 应只返回 TokenExhaustedError，实际 %v", host, err)
				}
				rejected++
				continue
			}
			tok.Succ() // 模拟下游请求成功
			passed++
		}
		if passed != int(initToken) {
			t.Fatalf("host %s 本秒应放行 %d 个，实际 %d", host, initToken, passed)
		}
		if rejected != reqPerHost-int(initToken) {
			t.Fatalf("host %s 本秒应拒绝 %d 个，实际 %d", host, reqPerHost-int(initToken), rejected)
		}
	}
}

// TestManagerAddOrInitAndGetDel 演示：Manager 基础 API，含依赖注入式配置与默认兜底。
func TestManagerAddOrInitAndGetDel(t *testing.T) {
	// 依赖注入式配置（等价于框架注入 initTokenNum/failLimit/triggerFailrate）
	m := &FtkManager{InitTokenNum: 50, InitFailLimit: 5, TriggerFailrate: 0.1}
	if err := m.Start(); err != nil {
		t.Fatalf("Start 失败: %v", err)
	}

	ft := m.AddOrInitDefault("svc-a")
	if ft == nil {
		t.Fatal("AddOrInitDefault 应返回非 nil 桶")
	}
	if ft.initTokenNum != 50 || ft.failLimit != 5 || ft.triggerFailrate != 0.1 {
		t.Fatalf("应使用注入的参数，实际 init=%d failLimit=%d trigger=%v",
			ft.initTokenNum, ft.failLimit, ft.triggerFailrate)
	}

	// 未调 Start 时 AddOrInitDefault 也应能工作（参数走包级默认值）
	m2 := &FtkManager{}
	ft2 := m2.AddOrInitDefault("svc-b")
	if ft2.initTokenNum != init_token_num || ft2.failLimit != fail_limit {
		t.Fatalf("未 Start 时应用默认参数，实际 init=%d failLimit=%d", ft2.initTokenNum, ft2.failLimit)
	}

	// Get 命中与未命中
	if m.Get("svc-a") != ft {
		t.Fatal("Get 应返回同一个实例")
	}
	if m.Get("not-exist") != nil {
		t.Fatal("Get 不存在的 host 应返回 nil")
	}

	// Del 删除
	m.Del("svc-a")
	if m.Get("svc-a") != nil {
		t.Fatal("Del 后 Get 应返回 nil")
	}
}
