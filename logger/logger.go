package logger

import (
	"fmt"
	"github.com/alecthomas/log4go"
	"runtime"
)

var Logger log4go.Logger

const (
	Depth = 1
)

func init() {
	Logger = log4go.NewLogger()
}

func LoadConfiguration(path string) {
	Logger.LoadConfiguration(path)
}

// log4go默认使用异步写入，如果主程序结束过快，日志可能来不及写入。
// 务必使用defer log4go.Close()或在main函数结束时调用log4go.Close()
func Close() {
	Logger.Close()
}
func Debug(arg0 interface{}, args ...interface{}) {
	if arg0Str, ok := arg0.(string); ok{
		arg0 = genArg0WithDepth(Depth+1, arg0Str)
	}
	Logger.Debug(arg0, args...)
}

func Info(arg0 interface{}, args ...interface{}) {
	if arg0Str, ok := arg0.(string); ok{
		arg0 = genArg0WithDepth(Depth+1, arg0Str)
	}
	Logger.Info(arg0, args...)
}

func Warn(arg0 interface{}, args ...interface{}) {
	if arg0Str, ok := arg0.(string); ok{
		arg0 = genArg0WithDepth(Depth+1, arg0Str)
	}
	Logger.Warn(arg0, args...)
}

func Error(arg0 interface{}, args ...interface{}) {
	if arg0Str, ok := arg0.(string); ok{
		arg0 = genArg0WithDepth(Depth+1, arg0Str)
	}
	Logger.Error(arg0, args...)
}

func Critical(arg0 interface{}, args ...interface{}) {
	if arg0Str, ok := arg0.(string); ok{
		arg0 = genArg0WithDepth(Depth+1, arg0Str)
	}
	Logger.Critical(arg0, args...)
}

// 根据调用栈深度生成arg0，附带上文件名、行号
func genArg0WithDepth(depth int, arg0 string) string {
	pc, _, line, ok := runtime.Caller(depth)
	var src string
	if ok {
		src = runtime.FuncForPC(pc).Name() + ":" + fmt.Sprintf("%v", line)
	}
	return fmt.Sprintf("[%v] %v", src, arg0)
}