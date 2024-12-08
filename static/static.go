/**
 * statik工具，可以将一些静态的文件，比如一整个目录，都编译成一个go文件，这样就不用去读取遍历目录文件了
 * 比如：嵌入web前端资源, 如js/css/html, 用于将包含了web页面的项目通过一个go可执行文件发布出去
 * 或者嵌入template模板文件, 这样发布将很简单, 不再需要将tpl文件复制到服务器
 *
 * 安装：
 *   go get github.com/rakyll/statik
 *   go install github.com/rakyll/statik
 *
 * 使用：
 *   1. 首先搞一个目录，并且里面有很多静态文件，比如这里的cnf（注意要有扩展名，否则读不到）
 *   2. 将这个cnf目录，编译成一个go文件，执行如下命令
 *       statik -src=./cnf -f # 这里直接用相对路径./cnf即可，因为实际
 *       这个命令将会生成一个./statik的文件夹，这个就是生成的go文件，需要将它在代码中导入
 *   3. 使用statik就可以读取到文件的内容，并且，所有文件的路径都是相cnf的绝对路径
 */
package static

import (
	"fmt"
	_ "github.com/hq-cml/go-tools/static/statik" // 这里就是要导入生成的go代码
	"github.com/rakyll/statik/fs"
	"io/ioutil"
)

// path的值，是相对于cnf的绝对路径！
// /a.json
// /dir1/dir2/b.json
func UseStatik(path string) {
	statikFS, err := fs.New()
	if err != nil {
		panic(err)
	}

	// Access individual files by their paths.
	r, err := statikFS.Open(path)
	if err != nil {
		panic(err)
	}
	defer r.Close()
	contents, err := ioutil.ReadAll(r)
	if err != nil {
		panic(err)
	}

	fmt.Println(string(contents))
}
