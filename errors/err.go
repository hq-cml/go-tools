/*
 * Golang 的 error 不会像 Java 那样打印 stackTrack 信息。回溯 err 非常不方便。
 * 如果自己层层log，写起来费劲并且高并发下还会被隔开。
 * 通过 github.com/pkg/errors 这个包来处理 err，可以解决这些问题。
 *
 * 如果只需要附带自己信息，可以WithMessage(err, msg)
 * 如果需要记录调用栈函数可以Wrap(err, msg)
 *     注意，使用 log.Errorf("%+v", err) 才会打印 stackTrack，使用 %v %s 不行
 *
 * 利用这个库，就可以不必处处打印错误日志，尤其是在某些第三方调用的底层
 * 将错误层层上报，在高层统一一次打印，使得问题定位更加清晰
 */
package errors

import (
    pkgerrors "github.com/pkg/errors"
    "os"
)

// 在错误上附带自己的信息
func genErr1() error {
    _, err := os.Open("noexist.txt")
    if err != nil {
        return pkgerrors.WithMessage(err, "genErr1")
    }
    return nil
}

// 错误栈包裹（打印需要 %+v）
func genErr2() error {
    _, err := os.Open("noexist.txt")
    if err != nil {
        return pkgerrors.Wrap(err, "genErr2")
    }
    return nil
}

func genErr3() error {
    return genErr2()
}