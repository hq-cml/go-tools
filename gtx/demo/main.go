package main

import (
	"fmt"
	"github.com/hq-cml/go-tools/gtx"
	"time"
)

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

// rename to run
func main2() {
	// 示例1：使用 GoWithGtx 启动安全的 goroutine
	fmt.Println("=== 示例1：基本使用 ===")
	
	for i := 0; i < 5; i++ {
		id := i
		gtx.GoWithGtx(func() {
			// 在当前 goroutine 的 gtx 中存储数据
			gtx.Set("worker_id", id)
			gtx.Set("start_time", time.Now().Unix())
			
			// 模拟工作
			time.Sleep(100 * time.Millisecond)
			
			// 读取数据
			if val, ok := gtx.Get("worker_id"); ok {
				fmt.Printf("Worker %d 完成工作\n", val)
			}
			
			// 退出时会自动调用 Clear4Current() 清理内存
		})
	}
	
	time.Sleep(500 * time.Millisecond)
	
	// 示例2：使用 GoWithGtxReturn 获取返回值
	fmt.Println("\n=== 示例2：带返回值 ===")
	
	result := gtx.GoWithGtxReturn(func() interface{} {
		gtx.Set("calculation", "done")
		time.Sleep(50 * time.Millisecond)
		return 42
	})
	
	val := <-result
	fmt.Printf("计算结果: %v\n", val)
	
	// 示例3：错误使用示范（会导致内存泄漏）
	fmt.Println("\n=== 示例3：错误使用（不推荐） ===")
	
	// 这种方式不会自动清理，可能导致内存泄漏！
	go func() {
		gtx.Init4Current()
		gtx.Set("key", "value")
		// 如果这里忘记调用 gtx.Clear4Current()，内存就会泄漏
		// defer gtx.Clear4Current()  // 别忘了这行！
	}()
	
	time.Sleep(100 * time.Millisecond)
	
	fmt.Println("\n=== 所有示例完成 ===")
}

