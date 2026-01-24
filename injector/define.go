package injector

import (
	orderMap "github.com/hq-cml/go-tools/order-map"
	"reflect"
	"sync"
)

// Graph 依赖注入图结构，用于管理对象的注册和查找
type Graph struct {
	l         sync.RWMutex         // 读写锁
	container *orderMap.OrderedMap // 对象容器，用一个插入顺序的Map存储注入的对象
	Logger    Logger
}

// Object 表示注入的对象
type Object struct {
	Name    string       // 对象名称
	refType reflect.Type // 对象的动态类型
	Value   interface{}  // 承载对象本身
	closed  bool         // 是否已关闭 ？？
}

// 注入接口定义
type Injectable interface {
	Startable
	Closeable
}

type Startable interface {
	Start() error
}

type Closeable interface {
	Close()
}

// 日志接口定义
type Logger interface {
	IsDebugEnabled() bool
	Debug(format interface{}, v ...interface{})
	Info(format interface{}, v ...interface{})
	Error(format interface{}, v ...interface{}) error
}
