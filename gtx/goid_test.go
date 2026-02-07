package gtx

import (
	"sync"
	"testing"
	"time"
)

// TestGetGoIdBasic 测试基本功能
func TestGetGoIdBasic(t *testing.T) {
	id := GetGoId()

	// ID 应该是正整数
	if id <= 0 {
		t.Errorf("GetGoId 应该返回正整数，实际返回 %d", id)
	}

	t.Logf("当前 goroutine ID: %d", id)
}

// TestGetGoIdConsistency 测试同一线程多次调用返回相同 ID
func TestGetGoIdConsistency(t *testing.T) {
	id1 := GetGoId()
	id2 := GetGoId()
	id3 := GetGoId()

	if id1 != id2 || id2 != id3 {
		t.Errorf("同一线程多次调用返回不同 ID: %d, %d, %d", id1, id2, id3)
	}
}

// TestGetGoIdUniqueness 测试不同 goroutine 有不同 ID
func TestGetGoIdUniqueness(t *testing.T) {
	const numGoroutines = 100

	ids := make(map[int]bool)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			id := GetGoId()

			mu.Lock()
			if ids[id] {
				t.Errorf("发现重复的 goroutine ID: %d", id)
			}
			ids[id] = true
			mu.Unlock()
		}()
	}

	wg.Wait()

	if len(ids) != numGoroutines {
		t.Errorf("期望 %d 个不同的 ID，实际得到 %d 个", numGoroutines, len(ids))
	}
}

// TestGetGoIdInNestedGoroutines 测试嵌套 goroutine
func TestGetGoIdInNestedGoroutines(t *testing.T) {
	mainId := GetGoId()

	var childId int
	done := make(chan bool)

	go func() {
		childId = GetGoId()
		done <- true
	}()

	<-done

	if childId == mainId {
		t.Error("父子 goroutine 的 ID 不应该相同")
	}

	if childId <= 0 {
		t.Error("子 goroutine 的 ID 应该是正整数")
	}
}

// TestGetGoIdWithGtx 测试与 gtx 结合使用
func TestGetGoIdWithGtx(t *testing.T) {
	defer Clear4Current()

	Init4Current()

	// 获取 ID
	id1 := GetGoId()

	// 存储数据
	Set("id", id1)

	// 再次获取 ID
	id2 := GetGoId()

	// 验证 ID 一致
	if id1 != id2 {
		t.Error("ID 不一致")
	}

	// 验证存储的 ID
	if stored, ok := Get("id"); !ok || stored != id1 {
		t.Error("存储的 ID 不匹配")
	}
}

// TestGetGoIdManyGoroutines 测试大量 goroutine
func TestGetGoIdManyGoroutines(t *testing.T) {
	const numGoroutines = 1000

	ids := make(chan int, numGoroutines)
	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ids <- GetGoId()
		}()
	}

	wg.Wait()
	close(ids)

	// 收集所有 ID
	idMap := make(map[int]int)
	for id := range ids {
		idMap[id]++
	}

	// 检查重复
	duplicates := 0
	for id, count := range idMap {
		if count > 1 {
			t.Errorf("ID %d 出现了 %d 次", id, count)
			duplicates++
		}
	}

	if duplicates > 0 {
		t.Errorf("发现 %d 个重复的 ID", duplicates)
	}

	t.Logf("成功创建了 %d 个不同 ID 的 goroutine", len(idMap))
}

// TestGetGoIdConcurrent 测试并发获取 ID
func TestGetGoIdConcurrent(t *testing.T) {
	const numWorkers = 50
	const iterations = 100

	var wg sync.WaitGroup
	errors := make(chan string, numWorkers*iterations)

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerId int) {
			defer wg.Done()

			myId := GetGoId()

			for j := 0; j < iterations; j++ {
				if currentId := GetGoId(); currentId != myId {
					errors <- "Worker ID 不一致"
					return
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	errCount := 0
	for range errors {
		errCount++
	}

	if errCount > 0 {
		t.Errorf("发现 %d 个错误", errCount)
	}
}

// TestGetGoIdWithTimeout 测试带超时的场景
func TestGetGoIdWithTimeout(t *testing.T) {
	done := make(chan int)

	go func() {
		id := GetGoId()
		done <- id
	}()

	select {
	case id := <-done:
		if id <= 0 {
			t.Error("获取的 ID 应该是正整数")
		}
	case <-time.After(time.Second):
		t.Error("获取 ID 超时")
	}
}

// TestGetGoIdSequential 测试顺序执行的 goroutine
func TestGetGoIdSequential(t *testing.T) {
	const count = 50

	ids := make([]int, 0, count)

	for i := 0; i < count; i++ {
		done := make(chan int)
		go func() {
			done <- GetGoId()
		}()

		id := <-done

		for _, existingId := range ids {
			if id == existingId {
				t.Errorf("顺序执行的 goroutine 产生了重复 ID: %d", id)
			}
		}

		ids = append(ids, id)
	}

	if len(ids) != count {
		t.Errorf("期望 %d 个 ID，实际 %d 个", count, len(ids))
	}
}

// BenchmarkGetGoId 基准测试
func BenchmarkGetGoId(b *testing.B) {
	for i := 0; i < b.N; i++ {
		GetGoId()
	}
}

// BenchmarkGetGoIdParallel 并发基准测试
func BenchmarkGetGoIdParallel(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			GetGoId()
		}
	})
}

// BenchmarkGetGoIdWithGtx 测试结合 gtx 的性能
func BenchmarkGetGoIdWithGtx(b *testing.B) {
	Init4Current()
	defer Clear4Current()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		id := GetGoId()
		Set("id", id)
		Get("id")
	}
}
