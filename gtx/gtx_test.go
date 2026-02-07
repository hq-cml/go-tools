package gtx

import (
	"sync"
	"testing"
	"time"
)

// TestSetAndGet 测试基本的 Set 和 Get 操作
func TestSetAndGet(t *testing.T) {
	defer Clear4Current()
	Init4Current()

	// 测试字符串
	Set("string_key", "hello")
	if val, ok := Get("string_key"); !ok || val != "hello" {
		t.Errorf("获取 string_key 失败，期望 hello，得到 %v", val)
	}

	// 测试整数
	Set("int_key", 42)
	if val, ok := Get("int_key"); !ok || val != 42 {
		t.Errorf("获取 int_key 失败，期望 42，得到 %v", val)
	}

	// 测试结构体
	type TestStruct struct {
		Name string
		Age  int
	}
	Set("struct_key", TestStruct{Name: "Alice", Age: 30})
	if val, ok := Get("struct_key"); !ok {
		t.Error("获取 struct_key 失败")
	} else if s, ok := val.(TestStruct); !ok || s.Name != "Alice" {
		t.Errorf("struct_key 值不匹配，得到 %v", val)
	}

	// 测试不存在的 key
	if _, ok := Get("non_existent_key"); ok {
		t.Error("不存在的 key 应该返回 false")
	}
}

// TestSetWithoutInit 测试未初始化时的行为
func TestSetWithoutInit(t *testing.T) {
	Clear4Current()

	// 未初始化时 Set 应该返回 false
	if Set("key", "value") {
		t.Error("未初始化时 Set 应该返回 false")
	}

	// 未初始化时 Get 应该返回 false
	if _, ok := Get("key"); ok {
		t.Error("未初始化时 Get 应该返回 false")
	}

	// 未初始化时 Del 应该返回 false
	if Del("key") {
		t.Error("未初始化时 Del 应该返回 false")
	}
}

// TestDel 测试删除功能
func TestDel(t *testing.T) {
	defer Clear4Current()
	Init4Current()

	Set("key1", "value1")
	Set("key2", "value2")

	// 删除 key1
	if !Del("key1") {
		t.Error("删除存在的 key 应该返回 true")
	}

	// 确认 key1 已删除
	if _, ok := Get("key1"); ok {
		t.Error("key1 应该已被删除")
	}

	// 确认 key2 仍存在
	if _, ok := Get("key2"); !ok {
		t.Error("key2 不应该被删除")
	}

	// 删除不存在的 key 应该返回 true（delete 操作不报错）
	if !Del("non_existent") {
		t.Error("删除不存在的 key 应该返回 true")
	}
}

// TestGetCurrCtx 测试获取完整上下文
func TestGetCurrCtx(t *testing.T) {
	defer Clear4Current()
	Init4Current()

	Set("k1", "v1")
	Set("k2", 123)

	ctx, ok := GetCurrCtx()
	if !ok {
		t.Fatal("GetCurrCtx 应该返回 true")
	}

	if len(ctx) != 2 {
		t.Errorf("上下文应该有 2 个 key，实际有 %d", len(ctx))
	}

	if ctx["k1"] != "v1" {
		t.Error("k1 的值不匹配")
	}

	if ctx["k2"] != 123 {
		t.Error("k2 的值不匹配")
	}
}

// TestIncr 测试计数器自增
func TestIncr(t *testing.T) {
	defer Clear4Current()
	Init4Current()

	// 首次自增，key 不存在
	prev, ok := Incr("counter", 5)
	if !ok {
		t.Error("首次 Incr 应该返回 true")
	}
	if prev != 0 {
		t.Errorf("首次 Incr 应该返回 0，实际返回 %d", prev)
	}

	// 验证值已设置
	if val, ok := Get("counter"); !ok || val != 5 {
		t.Errorf("counter 应该为 5，实际为 %v", val)
	}

	// 再次自增
	prev, ok = Incr("counter", 3)
	if !ok || prev != 5 {
		t.Errorf("第二次 Incr 应该返回 5，实际返回 %d", prev)
	}

	if val, ok := Get("counter"); !ok || val != 8 {
		t.Errorf("counter 应该为 8，实际为 %v", val)
	}
}

// TestIncrTypeMismatch 测试 Incr 类型不匹配的情况
func TestIncrTypeMismatch(t *testing.T) {
	defer Clear4Current()
	Init4Current()

	// 设置非整数类型
	Set("counter", "not_an_int")

	// 尝试自增，应该重置为 value
	prev, ok := Incr("counter", 10)
	if !ok {
		t.Error("类型不匹配时 Incr 应该返回 true")
	}
	if prev != 0 {
		t.Errorf("类型不匹配时应该返回 0，实际返回 %d", prev)
	}

	// 验证值被重置
	if val, ok := Get("counter"); !ok || val != 10 {
		t.Errorf("counter 应该被重置为 10，实际为 %v", val)
	}
}

// TestDecr 测试计数器自减
func TestDecr(t *testing.T) {
	defer Clear4Current()
	Init4Current()

	// 首次自减，key 不存在
	prev, ok := Decr("counter", 5)
	if !ok {
		t.Error("首次 Decr 应该返回 true")
	}
	if prev != 0 {
		t.Errorf("首次 Decr 应该返回 0，实际返回 %d", prev)
	}

	// 验证值为 -5
	if val, ok := Get("counter"); !ok || val != -5 {
		t.Errorf("counter 应该为 -5，实际为 %v", val)
	}

	// 再次自减
	prev, ok = Decr("counter", 3)
	if !ok || prev != -5 {
		t.Errorf("第二次 Decr 应该返回 -5，实际返回 %d", prev)
	}

	if val, ok := Get("counter"); !ok || val != -8 {
		t.Errorf("counter 应该为 -8，实际为 %v", val)
	}
}

// TestIncrDecrWithoutInit 测试未初始化时的计数器行为
func TestIncrDecrWithoutInit(t *testing.T) {
	Clear4Current()

	prev, ok := Incr("counter", 1)
	if ok {
		t.Error("未初始化时 Incr 应该返回 false")
	}
	if prev != 0 {
		t.Error("未初始化时 Incr 应该返回 0")
	}

	prev, ok = Decr("counter", 1)
	if ok {
		t.Error("未初始化时 Decr 应该返回 false")
	}
	if prev != 0 {
		t.Error("未初始化时 Decr 应该返回 0")
	}
}

// TestJsonCurrent 测试 JSON 导出
func TestJsonCurrent(t *testing.T) {
	defer Clear4Current()
	Init4Current()

	// 空上下文
	json1 := JsonCurrent()
	if json1 != "{}" {
		t.Errorf("空上下文的 JSON 应该是 {}，实际是 %s", json1)
	}

	// 添加数据
	Set("name", "Alice")
	Set("age", 30)

	json2 := JsonCurrent()
	if json2 == "{}" {
		t.Error("有数据时 JSON 不应该是 {}")
	}

	// 验证 JSON 包含预期内容
	if json2 != `{"age":30,"name":"Alice"}` && json2 != `{"name":"Alice","age":30}` {
		t.Errorf("JSON 格式不符合预期: %s", json2)
	}
}

// TestGoWithGtx 测试安全包装函数
func TestGoWithGtx(t *testing.T) {
	var wg sync.WaitGroup
	results := make(chan int, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		id := i
		GoWithGtx(func() {
			defer wg.Done()
			Set("worker_id", id)
			time.Sleep(10 * time.Millisecond)
			if val, ok := Get("worker_id"); ok {
				results <- val.(int)
			}
		})
	}

	wg.Wait()
	close(results)

	// 验证所有 goroutine 都完成了工作
	count := 0
	for range results {
		count++
	}
	if count != 10 {
		t.Errorf("期望 10 个结果，实际 %d", count)
	}

	// 验证数据已被清理（当前 goroutine 不应该有数据）
	if Exist4Current() {
		t.Error("GoWithGtx 应该清理数据")
	}
}

// TestGoWithGtxReturn 测试带返回值的安全包装函数
func TestGoWithGtxReturn(t *testing.T) {
	result := GoWithGtxReturn(func() interface{} {
		Set("calc", 1)
		return 42
	})

	val := <-result
	if val != 42 {
		t.Errorf("期望返回 42，实际 %v", val)
	}

	// 验证数据已被清理
	if Exist4Current() {
		t.Error("GoWithGtxReturn 应该清理数据")
	}
}

// TestConcurrency 测试并发安全性
func TestConcurrency(t *testing.T) {
	const numGoroutines = 100
	const numOperations = 100

	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			defer Clear4Current()
			Init4Current()

			// 每个 goroutine 进行多次操作
			for j := 0; j < numOperations; j++ {
				Set("id", id)
				Set("count", j)
				Incr("total", 1)
			}

			// 验证自己的数据没有被其他 goroutine 修改
			if val, ok := Get("id"); !ok || val != id {
				t.Errorf("Goroutine %d: id 被篡改，期望 %d，实际 %v", id, id, val)
			}

			if val, ok := Get("total"); !ok || val != numOperations {
				t.Errorf("Goroutine %d: total 不正确，期望 %d，实际 %v", id, numOperations, val)
			}
		}(i)
	}

	wg.Wait()
}

// TestMultipleInit 测试重复初始化不会覆盖数据
func TestMultipleInit(t *testing.T) {
	defer Clear4Current()

	Init4Current()
	Set("key", "value1")

	// 再次初始化
	Init4Current()

	// 数据应该还在
	if val, ok := Get("key"); !ok || val != "value1" {
		t.Error("重复初始化不应该清除已有数据")
	}
}

// TestIsolation 测试不同 goroutine 之间的隔离性
func TestIsolation(t *testing.T) {
	var wg sync.WaitGroup
	errors := make(chan string, 2)

	// Goroutine 1
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer Clear4Current()
		Init4Current()

		Set("data", "goroutine1")
		time.Sleep(50 * time.Millisecond) // 等待另一个 goroutine 设置

		if val, ok := Get("data"); !ok || val != "goroutine1" {
			errors <- "Goroutine 1: 数据被其他 goroutine 篡改"
		}
	}()

	// Goroutine 2
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer Clear4Current()
		Init4Current()

		Set("data", "goroutine2")

		if val, ok := Get("data"); !ok || val != "goroutine2" {
			errors <- "Goroutine 2: 数据被其他 goroutine 篡改"
		}
	}()

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Error(err)
	}
}

// BenchmarkSet 基准测试：Set 操作
func BenchmarkSet(b *testing.B) {
	Init4Current()
	defer Clear4Current()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Set("key", i)
	}
}

// BenchmarkGet 基准测试：Get 操作
func BenchmarkGet(b *testing.B) {
	Init4Current()
	defer Clear4Current()
	Set("key", "value")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Get("key")
	}
}

// BenchmarkIncr 基准测试：Incr 操作
func BenchmarkIncr(b *testing.B) {
	Init4Current()
	defer Clear4Current()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Incr("counter", 1)
	}
}

// BenchmarkConcurrentSet 基准测试：并发 Set
func BenchmarkConcurrentSet(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		Init4Current()
		defer Clear4Current()

		i := 0
		for pb.Next() {
			Set("key", i)
			i++
		}
	})
}
