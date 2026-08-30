package future

import (
	"bytes"
	"context"
	"errors"
	"log"
	"strings"
	"testing"
	"time"

	"github.com/hq-cml/go-tools/gtx"
)

func TestGet_Success(t *testing.T) {
	f := NewFuture(func() (Value, error) {
		return 42, nil
	})

	val, err := f.Get()
	if err != nil {
		t.Fatalf("期望 nil 错误，实际: %v", err)
	}
	if val != 42 {
		t.Fatalf("期望 42，实际: %v", val)
	}
}

func TestGet_Error(t *testing.T) {
	expectedErr := errors.New("任务失败")
	f := NewFuture(func() (Value, error) {
		return nil, expectedErr
	})

	val, err := f.Get()
	if err != expectedErr {
		t.Fatalf("期望 %v，实际: %v", expectedErr, err)
	}
	if val != nil {
		t.Fatalf("期望 nil，实际: %v", val)
	}
}

func TestGet_PanicRecover(t *testing.T) {
	f := NewFuture(func() (Value, error) {
		panic("炸了")
	})

	val, err := f.Get()
	if err == nil {
		t.Fatal("期望非 nil 错误，实际 nil")
	}
	if val != nil {
		t.Fatalf("期望 nil，实际: %v", val)
	}
	t.Logf("错误信息: %v", err)
}

func TestGetWithTimeout_Success(t *testing.T) {
	f := NewFuture(func() (Value, error) {
		time.Sleep(10 * time.Millisecond)
		return "ok", nil
	})

	val, err := f.GetWithTimeout(1 * time.Second)
	if err != nil {
		t.Fatalf("期望 nil 错误，实际: %v", err)
	}
	if val != "ok" {
		t.Fatalf("期望 ok，实际: %v", val)
	}
}

func TestGetWithTimeout_Timeout(t *testing.T) {
	f := NewFuture(func() (Value, error) {
		time.Sleep(2 * time.Second)
		return "slow", nil
	})

	val, err := f.GetWithTimeout(50 * time.Millisecond)
	if err != ErrTimeout {
		t.Fatalf("期望 ErrTimeout，实际: %v", err)
	}
	if val != nil {
		t.Fatalf("期望 nil，实际: %v", val)
	}
}

func TestGetWithContext_Success(t *testing.T) {
	f := NewFuture(func() (Value, error) {
		return true, nil
	})

	val, err := f.GetWithContext(context.Background())
	if err != nil {
		t.Fatalf("期望 nil 错误，实际: %v", err)
	}
	if val != true {
		t.Fatalf("期望 true，实际: %v", val)
	}
}

func TestGetWithContext_Canceled(t *testing.T) {
	f := NewFuture(func() (Value, error) {
		time.Sleep(2 * time.Second)
		return "slow", nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	val, err := f.GetWithContext(ctx)
	if err != context.Canceled {
		t.Fatalf("期望 context.Canceled，实际: %v", err)
	}
	if val != nil {
		t.Fatalf("期望 nil，实际: %v", val)
	}
}

func TestGetWithContext_DeadlineExceeded(t *testing.T) {
	f := NewFuture(func() (Value, error) {
		time.Sleep(2 * time.Second)
		return "slow", nil
	})

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	val, err := f.GetWithContext(ctx)
	if err != context.DeadlineExceeded {
		t.Fatalf("期望 context.DeadlineExceeded，实际: %v", err)
	}
	if val != nil {
		t.Fatalf("期望 nil，实际: %v", val)
	}
}

func TestGet_NilResult(t *testing.T) {
	f := NewFuture(func() (Value, error) {
		return nil, nil
	})

	val, err := f.Get()
	if err != nil {
		t.Fatalf("期望 nil 错误，实际: %v", err)
	}
	if val != nil {
		t.Fatalf("期望 nil，实际: %v", val)
	}
}

// 常规带参数的函数（利用闭包）
func myFunc(a int, b string) (int, string) {
	return a, b
}

func TestNormal_function(t *testing.T) {
	f := NewFuture(func() (Value, error) {
		m := make(map[string]interface{})
		v1, v2 := myFunc(1, "hello")
		m["v1"] = v1
		m["v2"] = v2
		return m, nil
	})
	val, err := f.Get()
	if err != nil {
		t.Fatalf("期望 nil 错误，实际: %v", err)
	}
	if val == nil {
		t.Fatalf("期望 nil，实际: %v", val)
	} else {
		t.Logf("val: %v", val)
	}
}

// TestFuture_WithLoggerAndGtxKeys 演示 WithLogger 和 WithGtxKeys 的完整用法：
//   - WithLogger: 注入自定义 Logger，捕获 Future 执行耗时日志
//   - WithGtxKeys: 除默认的 LogId/LaneTag 外，额外传递自定义 gtx key 到子 goroutine
func TestFuture_WithLoggerAndGtxKeys(t *testing.T) {
	// --- 准备自定义 Logger ---
	var buf bytes.Buffer
	logger := log.New(&buf, "", 0)

	// --- 在父 goroutine 中设置 gtx 上下文 ---
	gtx.Init4Current()
	defer gtx.Clear4Current()

	const customKey = "custom_trace_id"
	gtx.Set(LogId, "test-log-001")
	gtx.Set(LaneTag, "lane-A")
	gtx.Set(customKey, "trace-xyz-789")

	// --- 创建 Future，传入 WithLogger 和 WithGtxKeys ---
	ops := []Option{
		WithLogger(logger),
		WithGtxKeys(customKey), // 仅需要customKye，其他两个key是默认的
	}
	f := NewFuture(func() (Value, error) {
		// 子 goroutine 中读取传递过来的 gtx 值
		logIdVal, _ := gtx.Get(LogId)
		laneTagVal, _ := gtx.Get(LaneTag)
		customVal, customOk := gtx.Get(customKey)

		return map[string]interface{}{
			"logId":     logIdVal,
			"laneTag":   laneTagVal,
			"customVal": customVal,
			"customOk":  customOk,
		}, nil
	}, ops...)

	// --- 获取结果并验证 gtx 传递 ---
	val, err := f.Get()
	if err != nil {
		t.Fatalf("期望 nil 错误，实际: %v", err)
	}

	m, ok := val.(map[string]interface{})
	if !ok {
		t.Fatalf("期望 map 类型，实际: %T", val)
	}

	if m["logId"] != "test-log-001" {
		t.Errorf("期望 logId=test-log-001，实际: %v", m["logId"])
	}
	if m["laneTag"] != "lane-A" {
		t.Errorf("期望 laneTag=lane-A，实际: %v", m["laneTag"])
	}
	if m["customVal"] != "trace-xyz-789" {
		t.Errorf("期望 customVal=trace-xyz-789，实际: %v", m["customVal"])
	}
	if m["customOk"] != true {
		t.Error("期望 customOk=true")
	}

	// --- 验证 Logger 输出了耗时日志 ---
	out := buf.String()
	if !strings.Contains(out, "future cost:") {
		t.Fatalf("期望日志包含 'future cost:'，实际: %q", out)
	}
	t.Logf("gtx 传递结果: %v", m)
	t.Logf("日志输出: %s", out)
}
