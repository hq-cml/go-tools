package injector

import (
    "bytes"
    "encoding/json"
    "fmt"
    "os"
    "reflect"
    "strings"
    "time"

    "github.com/facebookgo/structtag"
    "github.com/hq-cml/go-tools/injector/implmap"
    orderMap "github.com/hq-cml/go-tools/order-map"
)

// String 返回对象的字符串表示
func (o Object) String() string {
    if o.refType.Kind() == reflect.Ptr {
        return fmt.Sprintf(`{"name":"%s","type":"%v","value":"%p"}`, o.Name, o.refType, o.Value)
    } else {
        return fmt.Sprintf(`{"name":"%s","type":"%v"}`, o.Name, o.refType)
    }
}

// NewGraph 创建新的依赖注入图
// 本质上它是一个Map，同时它的Key有两种模式：
//
//	refType => *Object
//	tagString => *Object  // 这里的tagString是`inject`
func newGraph() *Graph {
    g := &Graph{
        container: orderMap.NewOrderedMap(),
    }
    return g
}

// getTypeName 获取类型的完整名称，包括包路径和指针信息
// 这个值将作为在graph中查找对象
func getTypeName(t reflect.Type) string {
    isPtr := false
    if t.Kind() == reflect.Ptr {
        t = t.Elem()
        isPtr = true
    }
    var name string
    pkg := t.PkgPath()
    if pkg != "" {
        name = pkg + "." + t.Name()
    } else {
        name = t.Name()
    }
    if isPtr {
        name = "*" + name
    }
    return name
}

// find 根据名称，在graph中查找值（内部方法，不加锁）
func (g *Graph) find(name string) (*Object, bool) {
    f, ok := g.container.Get(name)
    if !ok {
        return nil, false
    }
    ret, ok := f.(*Object)
    if !ok {
        // g.named.Delete(name)
        panic(fmt.Sprintf("%s in graph is not a *Object, should not happen!", name))
        return nil, false
    } else {
        return ret, true
    }
}

// findByType 根据类型，在graph中查找值对象（内部方法，不加锁）
func (g *Graph) findByType(t reflect.Type) (*Object, bool) {
    n := getTypeName(t)
    return g.find(n)
}

// FindByType 根据类型查找对象
func (g *Graph) FindByType(t reflect.Type) (*Object, bool) {
    g.l.RLock()
    defer g.l.RUnlock()
    return g.findByType(t)
}

// _len 返回图中对象的数量（内部方法，不加锁）
func (g *Graph) _len() int {
    return g.container.Len()
}

// Len 返回图中对象的数量
func (g *Graph) Len() int {
    g.l.RLock()
    defer g.l.RUnlock()
    return g._len()
}

// Find 根据名称查找对象
// 这里的参数name应该是一个符合getTypeName()返回的值
func (g *Graph) Find(name string) (*Object, bool) {
    g.l.RLock()
    defer g.l.RUnlock()
    return g.find(name)
}

// del 删除指定名称的对象
func (g *Graph) del(name string) {
    g.container.Delete(name)
}

// set 设置指定名称的对象
func (g *Graph) set(name string, o *Object) {
    g.container.Set(name, o)
}

// setboth 同时按名称和类型设置对象
// 因为graph同时支持按照inject的tag和类型注册查找
func (g *Graph) setboth(name string, o *Object) {
    g.container.Set(name, o)
    if isStructPtr(o.refType) {
        tn := getTypeName(o.refType)
        g.container.Set(tn, o)
    }
}

// isZeroOfUnderlyingType 判断一个接口变量的底层类型是否为零值
func isZeroOfUnderlyingType(x interface{}) bool {
    if x == nil {
        return true
    }

    // 获取反射值 & 底层类型
    rvf := reflect.ValueOf(x)
    k := rvf.Kind()

    // 结构体，用DeepEqual比较与零值是否相等
    if k == reflect.Struct {
        return reflect.DeepEqual(reflect.New(reflect.TypeOf(x)).Elem().Interface(), x)
    }

    // 函数类型，检查是否为nil
    if k == reflect.Func {
        return rvf.IsNil()
    }

    // 指针、接口、通道、映射、切片等类型先检查是否为nil
    if (k == reflect.Ptr || k == reflect.Interface || k == reflect.Chan ||
        k == reflect.Map || k == reflect.Slice) && rvf.IsNil() {
        return true
    }

    // 数组、通道、映射、切片、字符串检查长度是否为0
    switch k {
    case reflect.Array, reflect.Chan, reflect.Map, reflect.Slice, reflect.String:
        if rvf.Len() <= 0 {
            return true
        } else {
            return false
        }
    }

    // 其他类型，通过reflect.Zero获取零值进行比较
    return x == reflect.Zero(reflect.TypeOf(x)).Interface()
}

// isNil 判断接口变量是否为nil
func isNil(v interface{}) bool {
    return reflect.ValueOf(v).IsNil()
}

// canNil 判断接口变量v是否可以为nil
// 也就是v的底层是否是一个指针或者接口，如果是则是canNil的
func canNil(v interface{}) bool {
    k := reflect.ValueOf(v).Kind()
    return k == reflect.Ptr || k == reflect.Interface
}

// register 核心实现：
// 注册一个对象到图中，支持依赖注入和单例模式
// 如果value是nil的，会自动化创建空结构，并且本逻辑是递归的
// 参数:
//
//	name      - 对象名称（通常来说，它是一个inject tag)，如果为空且value是结构体指针，则使用类型名
//	value     - 要注册的对象值
//	singleton - 是否为单例模式
//	skipFill  - 顶层意愿，是否跳过填充：
//	 如果是true，则跳过填充子字段（无需创建的时候）；
//	 如果为false，则无条件填充子字段
//
// 返回值:
//
//	interface{} - 注册后的对象
func (g *Graph) register(name string, value interface{}, singleton bool, skipFill bool) (interface{}, error) {
    // 获取对象反射类型
    reflectType := reflect.TypeOf(value)

    // 修正name值
    // 如果是结构体指针类型且名称为空，则自动获取类型名称，替代name
    // 如果不是结构体指针类型且名称为空，则返回错误（非结构体指针类型必须提供明确的名称）
    if isStructPtr(reflectType) {
        if name == "" {
            name = getTypeName(reflectType)
        }
    } else {
        if name == "" {
            return nil, fmt.Errorf("name can not be empty,name=%s,type=%v", name, reflectType)
        }
    }

    // 检查是否已注册
    found, ok := g.find(name)
    if ok {
        return nil, fmt.Errorf("already registered,name=%s,type=%v,found=%v", name, reflectType, found)
    }

    // 构建对象实体
    obj := &Object{
        Name:    name,
        refType: reflectType,
    }

    // 如果是结构体的指针，则会进入一个较为复杂的循环递归逻辑
    // 如果非结构体指针，则
    if isStructPtr(obj.refType) {
        // 获取实际指向的结构体的reflect.Type
        t := reflectType.Elem()
        var v reflect.Value
        needCreate := false // 是否需要动态创建
        // 如果是nil，则动态创建一个结构体
        if isNil(value) {
            needCreate = true
            v = reflect.New(t)
        } else {
            v = reflect.ValueOf(value)
        }

        // 到此：
        // t：是实际结构体的reflect.Type
        // v：是实际结构体的指针的reflect.Value，所以它需要v.Elem()才能真正得到结构体的reflect.Value

        // 遍历结构体字段进行依赖注入
        for i := 0; i < t.NumField(); i++ {
            // 如果不需要创建 && 顶层意愿是跳过填充=> 跳过填充
            if !needCreate && skipFill {
                continue
            }

            // 到此，需要创建 || 顶层意愿是不跳过填充 => 填充
            f := t.Field(i)
            vfe := v.Elem()
            vf := vfe.Field(i)

            // 到此：
            // f：具体字段的reflect.Type
            // vf：具体字段的reflect.Value

            // 如果字段已经填充，则跳过
            if vf.CanInterface() {
                if !isZeroOfUnderlyingType(vf.Interface()) {
                    continue
                }
            }

            // 如果字段是匿名字段 or 不可设置，则返回错误 TODO 测试（如果具体字段是一个指针？）
            if f.Anonymous || !vf.CanSet() {
                return nil, fmt.Errorf("inject injectTag must on a public field!field=%s,type=%s", f.Name, t.Name())
                // continue // useless code
            }

            // 抽取singleton的tag
            _, singletonStr, _ := structtag.Extract("singleton", string(f.Tag))
            singletonTag := false // 默认非单例
            if singletonStr == "true" {
                singletonTag = true
            }

            // 抽取cannil和nilable的tag，确定canNil
            _, canNilStr, _ := structtag.Extract("cannil", string(f.Tag))
            _, nilableStr, _ := structtag.Extract("nilable", string(f.Tag))
            canNil := false
            if canNilStr == "true" || nilableStr == "true" {
                canNil = true
            }

            // 抽取出inject的tag
            ok, injectTag, err := structtag.Extract("inject", string(f.Tag))
            if err != nil {
                return nil, fmt.Errorf("extract injectTag fail,f=%s,err=%v", f.Name, err)
            }
            if !ok {
                continue
            }

            // 如果inject非空，则按tag查找；否则按类型来查找
            var found *Object
            if injectTag != "" {
                // 先按照tag来查找，如果找不到且还是单例模式，则按照类型再给一次查找机会
                found, ok = g.find(injectTag)
                if !ok && singletonTag && isStructPtr(f.Type) {
                    found, ok = g.findByType(f.Type)
                }
            } else {
                found, ok = g.findByType(f.Type)
            }

            // 如果在当前graph中找不到，则递归得去动态注入
            if !ok || found == nil {
                // 如果允许为nil，说明容忍nil，则不会动态注入 TODO 测试
                if canNil {
                    continue
                }

                // 如果子字段本身是指向结构的指针，则动态创建一个结构体
                // 否则，需要依赖implmap去继续注册的流程
                if isStructPtr(f.Type) {
                    // 参照：Test_ReflectNewAt
                    // 这里用NewAt，封装出了一个空指针的Value然后还原interface{}
                    // 这就相当于创建了 var ptr *<T>，递归进去之后，再由里层去动态创建
                    // TODO 测试
                    _, err := g.register(injectTag, reflect.NewAt(f.Type.Elem(), nil).Interface(), singletonTag, skipFill)
                    if err != nil {
                        return nil, err
                    }
                } else {
                    // 到此，f.Type不是一个指针，那大概率是一个接口，需要去implmap中寻找
                    var implFound reflect.Type
                    impls := implmap.Get(injectTag)

                    // 注意implmap的结构，这里是一个list，尝试取第一个符合要求的
                    for _, impl := range impls {
                        if impl == nil {
                            continue
                        }
                        // 检查impl是否实现了目标接口
                        if impl.AssignableTo(f.Type) {
                            implFound = impl
                            break
                        }
                    }

                    // 在implmap中找到了符合要求的实现，则继续注册
                    // 根据implmap的实现，凡事登记进去的impl，必然是一个结构体指针
                    if implFound != nil {
                        _, err := g.register(injectTag, reflect.NewAt(implFound.Elem(), nil).Interface(), singletonTag, skipFill)
                        if err != nil {
                            return nil, err
                        }
                    } else {
                        return nil, fmt.Errorf("dependency field=%s,injectTag=%s not found in object %s:%v", f.Name, injectTag, name, reflectType)
                    }
                }

                // 到此处，递归已经结束，正常情况下injectTag已经应该成功注册，所以进行最终确认查找
                // ?? 这里的singleton为什么不是singletonTag，而是参数singleton，感觉不太对 ??
                if injectTag != "" {
                    found, ok = g.find(injectTag)
                    if !ok && singleton {
                        found, ok = g.findByType(f.Type)
                    }
                } else {
                    found, ok = g.findByType(f.Type)
                }
            }

            // 仍然没找到，说明尝试填充失败，则出现问题了，返回错误
            if !ok || found == nil {
                return nil, fmt.Errorf("dependency %s not found in object %s:%v", f.Name, name, reflectType)
            }

            // 这里有点晦涩，如果成功找到了，只能说明是在graph中动态注册了，但是实际上此时字段本身还是空着
            // 所以，需要设置vf（具体字段的reflect.Value）
            // 先类型检查，相当于是检查自动动态化注册的对象的类型，和目标字段的类型是否一致
            // 如果不一致，则需要抢救一下，比如对于数字系列，强转确保类型匹配
            reflectFoundValue := reflect.ValueOf(found.Value)
            if !found.refType.AssignableTo(f.Type) {
                // 处理类型转换，如不同大小的整数或浮点数类型
                switch reflectFoundValue.Kind() {
                case reflect.Int:
                    fallthrough
                case reflect.Int8:
                    fallthrough
                case reflect.Int16:
                    fallthrough
                case reflect.Int32:
                    fallthrough
                case reflect.Int64: // 所有的int系列，都按照int64处理
                    iv := reflectFoundValue.Int()
                    switch f.Type.Kind() {
                    case reflect.Int:
                        fallthrough
                    case reflect.Int8:
                        fallthrough
                    case reflect.Int16:
                        fallthrough
                    case reflect.Int32:
                        fallthrough
                    case reflect.Int64:
                        vf.SetInt(iv)
                    default:
                        return nil, fmt.Errorf("dependency name=%s,type=%v not valid in object %s:%v", f.Name, f.Type, name, reflectType)
                    }
                case reflect.Float32:
                    fallthrough
                case reflect.Float64: // 所有的float系列，都按照float64处理
                    fv := reflectFoundValue.Float()
                    switch f.Type.Kind() {
                    case reflect.Float32:
                        fallthrough
                    case reflect.Float64:
                        vf.SetFloat(fv)
                    default:
                        return nil, fmt.Errorf("dependency name=%s,type=%v not valid in object %s:%v", f.Name, f.Type, name, reflectType)
                    }
                default:
                    return nil, fmt.Errorf("dependency name=%s,type=%v not valid in object %s:%v", f.Name, f.Type, name, reflectType)
                }
            } else {
                vf.Set(reflectFoundValue)
            }
        }
        // 到这里，整个结构体的各个字段已经递归动态创建完毕了
        // 这个结构体自身也要填充到Object的Value中
        obj.Value = v.Interface()
    } else {
        // 说明待注册的实体非结构体指针，那么它可能是一个结构体，一个普通变量。。。
        // 先判断：如果canNil(是指针或者接口类型），则不能为nil，否则报错
        // 直接赋值value
        // TODO：
        // if inejection type is a struct(not a pointer), we should create a new struct every time when a inject tag is found
        // and no *Object is created and the created struct should NOT be set into graph.named(or there will be a memory leak)!
        // same as a bean's prototype scope in spring
        // otherwise all inject dependency will behave like spring's singleton bean scope
        // 如果注入类型是结构体（非指针），当使用inject标签时，应该每次创建新结构体
        // 同时，不应该创建*Object，并且创建的结构体不应该被注册到graph.named中，否则将造成内存泄漏
        // 就和Spring中的bean原型作用域一样
        // 否则，所有注入的依赖都将表现得像Spring的单例bean作用域
        if canNil(value) && isNil(value) {
            return nil, fmt.Errorf("register nil on name=%s, val=%v", name, value)
        }
        obj.Value = value
    }

    // 依赖解析完成，如果对象注册了Start钩子，则拉起Start()初始化对象
    toStart, ok := obj.Value.(Startable)
    if ok {
        st := time.Now()
        err := toStart.Start()
        cost := time.Now().Sub(st)
        // 初始化过久，则打印日志进行标记
        if cost > 5*time.Second {
            errMsg := fmt.Sprintf("obj start took too long,name=%v,time=%v,err=%v", name, cost, err)
            fmt.Fprint(os.Stderr, errMsg+"\n")
            if g.Logger != nil {
                g.Logger.Error(errMsg)
            }
        }
        // 初始化出错，则返回错误
        if err != nil {
            return nil, fmt.Errorf("Start object fail,name=%v,err=%v", name, err)
        }
    }

    // 注册到图中
    // （如果是结构体指针且单例模式，则同时注册tag和type）
    if isStructPtr(reflectType) && singleton {
        g.setboth(name, obj)
    } else {
        g.set(name, obj)
    }
    if g.Logger != nil && g.Logger.IsDebugEnabled() {
        toLogJson, toLogErr := json.Marshal(obj.Value)
        g.Logger.Debug("registered!name=%s,t=%v,v=%v,jsonerr=%v", name, reflectType, string(toLogJson), toLogErr)
        fmt.Fprint(os.Stdout, fmt.Sprintf("registered!name=%s,t=%v,v=%v\n", name, reflectType, obj.Value))
    } else {
        fmt.Fprint(os.Stdout, fmt.Sprintf("registered!name=%s,t=%v,v=%v\n", name, reflectType, obj.Value))
    }

    return obj.Value, nil
}

// RegisterNoFill 注册对象但不进行依赖注入
func (g *Graph) RegisterNoFill(name string, value interface{}) (interface{}, error) {
    g.l.Lock()
    defer g.l.Unlock()
    return g.register(name, value, false, true)
}

// RegWithoutInjection 注册对象但不进行依赖注入
func (g *Graph) RegWithoutInjection(name string, value interface{}) interface{} {
    return g.RegisterOrFailNoFill(name, value)
}

// RegisterOrFailNoFill 注册对象但不进行依赖注入，失败时panic
func (g *Graph) RegisterOrFailNoFill(name string, value interface{}) interface{} {
    v, err := g.RegisterNoFill(name, value)
    if err != nil {
        if g.Logger != nil {
            g.Logger.Error(err)
        }
        panic(fmt.Sprintf("reg fail,name=%v,err=%v", name, err.Error()))
    }
    return v
}

// RegisterOrFailSingleNoFill 注册单例对象但不进行依赖注入，失败时panic
func (g *Graph) RegisterOrFailSingleNoFill(name string, value interface{}) interface{} {
    v, err := g.RegisterSingleNoFill(name, value)
    if err != nil {
        if g.Logger != nil {
            g.Logger.Error(err)
        }
        panic(fmt.Sprintf("reg fail,name=%v,err=%v", name, err.Error()))
    }
    return v
}

// RegisterOrFail 注册对象并进行依赖注入，失败时panic
func (g *Graph) RegisterOrFail(name string, value interface{}) interface{} {
    v, err := g.Register(name, value)
    if err != nil {
        if g.Logger != nil {
            g.Logger.Error(err)
        }
        panic(fmt.Sprintf("reg fail,name=%v,err=%v", name, err.Error()))
    }
    return v
}

// RegisterOrFailSingle 注册单例对象并进行依赖注入，失败时panic
func (g *Graph) RegisterOrFailSingle(name string, value interface{}) interface{} {
    v, err := g.RegisterSingle(name, value)
    if err != nil {
        if g.Logger != nil {
            g.Logger.Error(err)
        }
        panic(fmt.Sprintf("reg fail,name=%v,err=%v", name, err.Error()))
    }
    return v
}

// RegisterSingleNoFill 注册单例对象但不进行依赖注入
func (g *Graph) RegisterSingleNoFill(name string, value interface{}) (interface{}, error) {
    g.l.Lock()
    defer g.l.Unlock()
    return g.register(name, value, true, true)
}

// Register 注册对象并进行依赖注入
func (g *Graph) Register(name string, value interface{}) (interface{}, error) {
    g.l.Lock()
    defer g.l.Unlock()
    return g.register(name, value, false, false)
}

// RegisterSingle 注册单例对象并进行依赖注入
func (g *Graph) RegisterSingle(name string, value interface{}) (interface{}, error) {
    g.l.Lock()
    defer g.l.Unlock()
    return g.register(name, value, true, false)
}

// SPrint 返回图的字符串表示
func (g *Graph) SPrint() string {
    g.l.RLock()
    defer g.l.RUnlock()
    return g.sPrint()
}

// sPrint 返回图的字符串表示（内部方法，不加锁）
func (g *Graph) sPrint() string {
    ret := "["
    iter := g.container.IterFunc()
    count := g._len()
    i := 0
    for kv, ok := iter(); ok; kv, ok = iter() {
        str := fmt.Sprintf(`{"key":"%s","object":%s}`, fmt.Sprintf("%s", kv.Key), fmt.Sprintf("%s", kv.Value))

        ret = ret + str
        i++
        if i < count {
            ret = ret + ","
        }
    }
    ret = ret + "]"
    return ret
}

// SPrintTree 返回依赖树的字符串表示
func (g *Graph) SPrintTree() string {
    g.l.RLock()
    defer g.l.RUnlock()
    buf := bytes.NewBufferString("dependence tree:\n")

    len := g.container.Len()
    i := 0
    iter := g.container.IterFunc()
    for kv, ok := iter(); ok; kv, ok = iter() {
        head := "├── "
        if i == 0 {
            head = "┌── "
        } else if i == len-1 {
            head = "└── "
        }
        i++
        g.sPrintTree(head, kv.Value.(*Object), buf)
    }
    return buf.String()
}

// sPrintTree 递归打印依赖树
func (g *Graph) sPrintTree(path string, o *Object, buf *bytes.Buffer) error {
    value := ""
    if o.refType.Kind() == reflect.Ptr {
        value = fmt.Sprintf("%p", o.Value)
    } else {
        value = fmt.Sprintf("%v", o.Value)
    }
    show := fmt.Sprintf("%s%s(%v=%v)\n", path, o.Name, o.refType, value)
    buf.WriteString(show)

    if isStructPtr(o.refType) {
        childPath := path
        childPath = strings.Replace(childPath, "└", " ", -1)
        childPath = strings.Replace(childPath, "├", "│", -1)
        childPath = strings.Replace(childPath, "─", " ", -1)
        childPath = strings.Replace(childPath, "┌", "│", -1)

        t := o.refType.Elem()

        // load tags of injected child
        var tags []string
        for i := 0; i < t.NumField(); i++ {
            structFiled := t.Field(i)
            ok, tag, err := structtag.Extract("inject", string(structFiled.Tag))
            if err != nil {
                return fmt.Errorf("extract tag fail,f=%s,err=%v", structFiled.Name, err)
            }
            if !ok {
                continue
            }

            if len(tag) == 0 {
                tag = getTypeName(structFiled.Type)
            }

            _, ok = g.find(tag)
            if ok {
                tags = append(tags, tag)
            }
        }

        for i, tag := range tags {
            corner := ""
            if i == len(tags)-1 {
                corner = childPath + " └── "
            } else {
                corner = childPath + " ├── "
            }
            childObject, _ := g.find(tag)
            g.sPrintTree(corner, childObject, buf)
        }
    }

    return nil
}

// beaware of the close order when use g.Close!
// every *Object will be Closed on reverse order
// of the Register
// there should be no defer xx.Close betwen g.Register
// function calls in main.exe
// Close 关闭图中的所有对象，按照注册的逆序关闭
func (g *Graph) Close() {
    g.l.Lock()
    defer g.l.Unlock()

    if g.Logger != nil {
        g.Logger.Info("close objects %v", g.sPrint())
    }
    var keys []string
    iter := g.container.RevIterFunc()
    for kv, ok := iter(); ok; kv, ok = iter() {
        k, ok := kv.Key.(string)
        if !ok {
            continue
        }
        keys = append(keys, k)
        o, ok := kv.Value.(*Object)
        if !ok {
            continue
        }
        if o.closed {
            continue
        }
        if isStructPtr(o.refType) {
            keys = append(keys, getTypeName(o.refType))
        }
        if o.Value == nil {
            continue
        }
        c, ok := o.Value.(Closeable)
        if ok {
            c.Close()
            if g.Logger != nil {
                g.Logger.Debug("closed!object=%s", o)
            }
            o.closed = true
        }
    }

    for _, k := range keys {
        g.del(k)
    }
    if g.Logger != nil {
        g.Logger.Info("inject graph closed all")
    }
}

// isStructPtr 判断类型是否为结构体指针
func isStructPtr(t reflect.Type) bool {
    return t.Kind() == reflect.Ptr && t.Elem().Kind() == reflect.Struct
}

// ReflectRegFields 反射注册结构体字段
func ReflectRegFields(v interface{}) map[string]interface{} {
    ret := make(map[string]interface{})

    vfe := reflect.ValueOf(v).Elem()
    t := vfe.Type()
    if t.Kind() != reflect.Struct {
        panic(fmt.Sprintf("value must be a struct pointer %v %v", v, t))
    }
    for i := 0; i < t.NumField(); i++ {
        f := t.Field(i)
        vf := vfe.Field(i)

        name := f.Name
        name = strings.ToLower(string(name[0])) + name[1:]
        vi := vf.Interface()

        Reg(name, vi)
        ret[name] = vi
    }
    return ret
}
