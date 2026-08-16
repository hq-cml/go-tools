package gtx

import (
	"fmt"
	goid "github.com/petermattis/goid"
	"runtime"
	"strconv"
	"strings"
)

// TODO 如果需要改变，后期通过注入变量的形式实现
const (
	DefaultUse = 2
)

// GetGoId 。两种实现思路
// 方案1：在runtime的Stack中，获取当前goroutine的ID，存在兼容性隐患且性能差
// 方案2：使用 github.com/petermattis/goid 库获取当前goroutine的ID
//       在amd64等主流架构上通过汇编直接读取g.goid字段，性能远优于GetGoId的runtime.Stack方案
func GetGoId() int {
	if DefaultUse == 1 {
		var buf [128]byte
		n := runtime.Stack(buf[:], false)
		// 提取 goroutine 后的数字
		idField := strings.Fields(strings.TrimPrefix(string(buf[:n]), "goroutine "))[0]
		id, err := strconv.Atoi(idField)
		if err != nil {
			panic(fmt.Sprintf("cannot get goroutine id: %v", err))
		}
		return id
	} else {
		return int(goid.Get())
	}
}
