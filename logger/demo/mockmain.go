package main

import (
	log "github.com/hq-cml/go-tools/logger"
	"github.com/hq-cml/go-tools/logger/demo/lib"
)

// 模拟Main函数
func main() {
	log.LoadConfiguration(`log.xml`)
	defer log.Close()
	lib.MyFuc4()
	MyFuc1()
}

func MyFuc1() {
	log.Info("MyFuc1 is called:%v", "foo")
	MyFuc2()
}

func MyFuc2() {
	//log.Info("MyFuc2 is called:%v", "bar")
	log.Error("MyFuc2 is called")
	lib.MyFuc3("hello")
}
