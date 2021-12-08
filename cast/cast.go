/*
 * 在Go中处理动态数据时，通常需要将数据从一种类型转换为另一种类型。Cast是一个库，以一致和简单的方式在不同的go类型之间转换。
 * Cast强制转换提供了一些ToXXX 的方法。这些方法将始终返回所需的类型。如果提供的输入不能转换为该类型，则返回该类型的0或nil值。
 * Cast也提供了 ToXXXE相同的方法。这些方法返回与ToXXX方法相同的结果，外加一个额外的错误，告诉您是否成功转换。
 * 使用这些方法，您可以分辨输入匹配零值时的不同，以及转换失败时返回零值时的不同。
 */
package cast

import (
	"fmt"
	"github.com/spf13/cast"
)

func UseCast() {
	fmt.Println(cast.ToString("mayonegg")) // "mayonegg"
	fmt.Println(cast.ToString(8) )    // "8"
	fmt.Println(cast.ToString(8.31))    // "8.31"
	fmt.Println(cast.ToString([]byte("one time"))) // "one time"
	fmt.Println(cast.ToString(nil))    // ""
	var foo interface{} = "one more time"
	fmt.Println(cast.ToString(foo))    // "one more time"
	fmt.Println("---------------------------------")

	fmt.Println(cast.ToInt(8))    // 8
	fmt.Println(cast.ToInt(8.31))    // 8
	fmt.Println(cast.ToInt("8"))    // 8
	fmt.Println(cast.ToInt(true))    // 1
	fmt.Println(cast.ToInt(false))    // 0
	var eight interface{} = 8
	fmt.Println(cast.ToInt(eight))    // 8
	fmt.Println(cast.ToInt(nil))    // 0
	fmt.Println("---------------------------------")

	fmt.Println(cast.ToInt("abc"))
	fmt.Println(cast.ToIntE("abc"))
}