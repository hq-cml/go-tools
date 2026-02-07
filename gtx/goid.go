package gtx

import (
	"fmt"
	"runtime"
	"strconv"
	"strings"
	// "github.com/v2pro/plz/gls"
)

// GetGoId 在runtime的Stack中，获取当前goroutine的ID
func GetGoId() int {
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	fmt.Println(string(buf[:n]))
	// 提取 goroutine 后的数字
	idField := strings.Fields(strings.TrimPrefix(string(buf[:n]), "goroutine "))[0]
	id, err := strconv.Atoi(idField)
	if err != nil {
		panic(fmt.Sprintf("cannot get goroutine id: %v", err))
	}
	return id
}

// 开源版本
// import "github.com/v2pro/plz/gls"
// func GetGoroutineID() int {
// 	return int(gls.GoID())
// }

// 对比测试
// func main() {
// 	for i := 0; i < 1000; i++ {
// 		go func() {
// 			if gtx.GetGoroutineID() != gtx.GetGoId() {
// 				panic("goroutine id != goid")
// 			}
// 			fmt.Println(gtx.GetGoroutineID(), gtx.GetGoId())
// 		}()
// 	}
// 	fmt.Println(gtx.GetGoroutineID(), gtx.GetGoId())
// 	time.Sleep(time.Second)
// }