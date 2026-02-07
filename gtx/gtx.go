package gtx

import (
	"encoding/json"
	"fmt"
	cmap "github.com/orcaman/concurrent-map"
)

type Gtx struct {
	goCtxMap cmap.ConcurrentMap
}

var _gtx *Gtx

func init() {
	_gtx = &Gtx{
		goCtxMap: cmap.New(),
	}
}

func Init4Current() {
	goid := GetGoId()
	gtx, ok := _gtx.goCtxMap.Get(fmt.Sprint(goid))
	if !ok || gtx == nil {
		gtx = make(map[interface{}]interface{})
		_gtx.goCtxMap.Set(fmt.Sprint(goid), gtx)
	}
}

func Clear4Current() {
	goid := GetGoId()
	_gtx.goCtxMap.Remove(fmt.Sprint(goid))
}

func GetCurrCtx() (map[interface{}]interface{}, bool) {
	goid := GetGoId()
	m, ok := _gtx.goCtxMap.Get(fmt.Sprint(goid))
	if !ok {
		return nil, false
	}
	if gtx, ok := m.(map[interface{}]interface{}); ok {
		return gtx, true
	} else {
		return nil, false
	}
}

// 当前goroutine是否存在gtx，也就是是否被Init过
func Exist4Current() bool {
	_, ok := GetCurrCtx()
	return ok
}

func Get(key interface{}) (interface{}, bool) {
	gtx, ok := GetCurrCtx()
	if !ok {
		return nil, false
	}
	ret, ok := gtx[key]
	return ret, ok
}

func Set(key interface{}, value interface{}) bool {
	gtx, ok := GetCurrCtx()
	if !ok {
		return false
	}
	gtx[key] = value
	return true
}

func Del(key interface{}) bool {
	gtx, ok := GetCurrCtx()
	if !ok {
		return false
	}
	delete(gtx, key)
	return true
}

// key对应的值加value
// 如果key对应的值不存在，则初始化为0
// 返回自增之前的值
func Incr(key interface{}, value int) (int, bool) {
	gtx, ok := GetCurrCtx()
	if !ok {
		return 0, false
	}
	v, ok := gtx[key]
	if !ok {
		gtx[key] = value
		return 0, true
	}
	vc, ok := v.(int)
	if !ok {
		gtx[key] = value
		return 0, true
	}
	gtx[key] = vc + value
	return vc, true
}

// key对应的值减value
// 如果key对应的值不存在，则初始化为0
// 返回自减之前的值
func Decr(key interface{}, value int) (int, bool) {
	gtx, ok := GetCurrCtx()
	if !ok {
		return 0, false
	}
	v, ok := gtx[key]
	if !ok {
		gtx[key] = -value
		return 0, true
	}
	vc, ok := v.(int)
	if !ok {
		gtx[key] = -value
		return 0, true
	}
	gtx[key] = vc - value
	return vc, true
}

func JsonCurrent() string {
	gtx, ok := GetCurrCtx()
	if !ok {
		return "{}"
	}
	// 转换为 map[string]interface{} 以便 JSON 序列化
	converted := make(map[string]interface{})
	for k, v := range gtx {
		if keyStr, ok := k.(string); ok {
			converted[keyStr] = v
		} else {
			converted[fmt.Sprintf("%v", k)] = v
		}
	}
	s, err := json.Marshal(converted)
	if err != nil {
		return "{}"
	}
	return string(s)
}

// GoWithGtx 安全地启动一个带有gtx的goroutine
// 自动处理Init和Clear，避免内存泄漏
func GoWithGtx(fn func()) {
	go func() {
		defer Clear4Current()
		Init4Current()
		fn()
	}()
}

// GoWithGtxReturn 安全地启动一个带有gtx的goroutine，支持返回值
// 自动处理Init和Clear，避免内存泄漏
func GoWithGtxReturn(fn func() interface{}) chan interface{} {
	result := make(chan interface{}, 1)
	go func() {
		defer Clear4Current()
		Init4Current()
		result <- fn()
		close(result)
	}()
	return result
}
