package lib

import (
	log "github.com/hq-cml/go-tools/logger"
)

func MyFuc3(name string) {
	log.Info("MyFuc3 is called:%d, args:%v", 123, name)
}

func MyFuc4() {
	log.Critical("MyFuc4 is called:%d", 123)
}