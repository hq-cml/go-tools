package logger

import (
	"github.com/alecthomas/log4go"
)

var Logger log4go.Logger

func init() {
	Logger = log4go.NewLogger()
}

func LoadConfiguration(path string) {
	Logger.LoadConfiguration(path)
}

func Close() {
	Logger.Close()
}

func AddFilter(name string, level log4go.Level, writer log4go.LogWriter) {
	Logger.AddFilter(name, level, writer)
}

func Info(arg0 interface{}, args ...interface{}) {
	Logger.Info(arg0, args)
}