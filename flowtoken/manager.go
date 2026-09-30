package flowtoken

import (
	log "github.com/hq-cml/go-tools/logger"
	"reflect"
	"sync"

	"github.com/hq-cml/go-tools/injector/implmap"
)

// init 将 FtkManager 类型注册到 implmap，支持通过依赖注入框架实例化
func init() {
	implmap.Add("flowTokenBucketManager", reflect.TypeOf((*FtkManager)(nil)))
}

// FlowTokenBucketManagerIntfs 管理器接口
// 通过host维度集中管理多个FlowToken实例，支持mock测试
type FlowTokenBucketManagerIntfs interface {
	AddOrInit(host string, initTokenNum int64, initFailLimit int64, triggerFailrate float64) *FlowTokenBucket
	AddOrInitDefault(host string) *FlowTokenBucket
	Get(host string) *FlowTokenBucket
	Del(host string)
}

// FtkManager FlowToken管理器
// 按 host 维度管理多个 FlowToken 实例，使用 sync.Map 保证并发安全
type FtkManager struct {
	// inject 标签支持依赖注入，cannil:"true" 表示可以为空
	InitFailLimit   int64   `inject:"flowTokenManagerInitFailLimit" cannil:"true"`
	InitTokenNum    int64   `inject:"flowTokenManagerInitTokenNum" cannil:"true"`
	TriggerFailrate float64 `inject:"flowTokenManagerTriggerFailrate" cannil:"true"`

	fts sync.Map // key=host(string), value=*FlowToken
}

// Start 管理器启动，初始化默认参数
// 当注入值无效时使用包级默认值
func (m *FtkManager) Start() error {
	m.InitTokenNum, m.InitFailLimit, m.TriggerFailrate = normalizeInit(m.InitTokenNum, m.InitFailLimit, m.TriggerFailrate)
	log.Info("flow token manager start up %v", m)
	return nil
}

// Close 管理器关闭，遍历所有FlowToken输出最终状态快照
func (m *FtkManager) Close() {
	m.fts.Range(func(key, val interface{}) bool {
		vv, ok := val.(*FlowTokenBucket)
		if ok && vv != nil {
			v := vv.GetSnapshot()
			log.Info("flow token manager close k=%v,v=%v", key, v)
		}
		return true
	})
}

// Get 获取指定host的FlowToken，不存在返回nil
func (m *FtkManager) Get(host string) *FlowTokenBucket {
	if ft, ok := m.fts.Load(host); ok {
		return ft.(*FlowTokenBucket)
	}
	return nil
}

// Del 删除指定host的FlowToken
func (m *FtkManager) Del(host string) {
	m.fts.Delete(host)
}

// AddOrInit 使用指定参数添加FlowToken

func (m *FtkManager) AddOrInit(host string, initTokenNum int64, initFailLimit int64, triggerFailrate float64) *FlowTokenBucket {
	return m.creatBucket(host, initTokenNum, initFailLimit, triggerFailrate)
}

// AddOrInitDefault 使用管理器默认参数添加FlowToken
func (m *FtkManager) AddOrInitDefault(host string) *FlowTokenBucket {
	return m.creatBucket(host, m.InitTokenNum, m.InitFailLimit, m.TriggerFailrate)
}

// creatBucket 创建FlowToken（幂等）
// 参数:
//   - host: 下游服务实例标识
//   - initTokenNum: 初始令牌数
//   - initFailLimit: 失败阈值
//   - triggerFailrate: 触发失败率，0表示使用默认值
func (m *FtkManager) creatBucket(host string, initTokenNum int64, initFailLimit int64, triggerFailrate float64) *FlowTokenBucket {
	if ft, ok := m.fts.Load(host); ok {
		return ft.(*FlowTokenBucket)
	}
	ft := NewFlowTokenIdInit(host, initTokenNum, initFailLimit, triggerFailrate)
	actual, loaded := m.fts.LoadOrStore(host, ft)
	if loaded {
		return actual.(*FlowTokenBucket)
	}
	log.Info("flowtoken manager add,h=%s,init=%d,failLimit=%d,triggerFailrate=%.2f", host, initTokenNum, initFailLimit, triggerFailrate)
	return ft
}
