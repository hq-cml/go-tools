package mock_main

import (
	"testing"
	log "github.com/hq-cml/go-tools/logger"
)

// 模拟Main函数
func TestMockMain(t *testing.T) {
	log.LoadConfiguration("log.xml")
	defer log.Close()
	MyFuc1()
}

func MyFuc1() {
	log.Info("MyFuc1 is called")
	MyFuc2()
}

func MyFuc2() {
	log.Info("MyFuc2 is called")
}
