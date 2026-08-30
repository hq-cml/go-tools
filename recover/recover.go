package recover

import (
	"fmt"

	easyerrors "github.com/go-errors/errors"
)

func WithRecover(fn func(), errHandler func(interface{})) (err interface{}) {
	defer func() {
		if err = recover(); err != nil {
			// easyerrors.Wrap(err, 2)：
			// 用 go-errors 库包装 panic 值，参数 2 表示跳过 2 层栈帧（Wrap 自身 + 当前 defer 层），从而定位到真正 panic 的调用处。
			wraped := easyerrors.Wrap(err, 2)
			// wraped.ErrorStack()：从包装的错误中提取完整堆栈信息字符串，用于后续告警/日志输出。
			stacktrace := wraped.ErrorStack()

			//fmt.Fprintln(os.Stderr, "panic_recovered:", stacktrace)
			if errHandler != nil {
				errHandler(err)
			}
			err = fmt.Errorf("%v", stacktrace)
		}
	}()

	fn()
	return
}
