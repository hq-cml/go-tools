# Bug List - Go Injector 依赖注入框架

## 严重问题 (Critical)

### Bug 1: 全局变量 `_g` 的空指针风险
**文件**: global.go:9, 15-81  
**问题描述**: 全局变量 `_g` 初始为 nil，所有全局函数（Close, Register, Find 等）直接使用 `_g` 而不检查是否为 nil。如果用户忘记调用 InitDefault()，程序会 panic。  
**影响**: 程序崩溃  
**修复建议**: 在每个使用 `_g` 的函数中添加 nil 检查：
```go
func Close() {
    if _g == nil {
        return
    }
    _g.Close()
}
```

### Bug 2: InitDefault() 的并发安全问题
**文件**: global.go:11-13  
**问题描述**: InitDefault() 对全局变量 `_g` 的赋值没有锁保护。在多 goroutine 环境下并发调用，可能产生竞态条件。  
**影响**: 竞态条件，数据竞争  
**修复建议**: 使用 sync.Once 保护：
```go
var _once sync.Once

func InitDefault() {
    _once.Do(func() {
        _g = newGraph()
    })
}
```

### Bug 3: isNil() 函数 panic 风险
**文件**: injector.go:174-176  
**问题描述**: isNil() 函数对非指针、非接口类型调用 IsNil() 会导致 panic。虽然当前调用处有前置检查，但这是脆弱的约定。  
**影响**: 如果未来代码修改导致传入非指针/接口类型，会导致 panic  
**修复建议**: 添加类型安全检查：
```go
func isNil(v interface{}) bool {
    if v == nil {
        return true
    }
    rv := reflect.ValueOf(v)
    switch rv.Kind() {
    case reflect.Ptr, reflect.Interface, reflect.Map, 
         reflect.Slice, reflect.Chan, reflect.Func:
        return rv.IsNil()
    default:
        return false
    }
}
```

### Bug 4: canNil() 未正确处理 nil 接口
**文件**: injector.go:178-183  
**问题描述**: canNil() 函数没有检查 v 是否为 nil，直接调用 reflect.ValueOf(v).Kind()。如果 v 是 nil 接口，会返回 reflect.Invalid。  
**影响**: 对于 nil 接口返回不正确的结果  
**修复建议**:
```go
func canNil(v interface{}) bool {
    if v == nil {
        return true
    }
    k := reflect.ValueOf(v).Kind()
    return k == reflect.Ptr || k == reflect.Interface || 
           k == reflect.Map || k == reflect.Slice || 
           k == reflect.Chan || k == reflect.Func
}
```

---

## 逻辑问题 (Logic Issues)

### Bug 5: Close() 中 keys 重复添加
**文件**: injector.go:697-730  
**问题描述**: 在 Close() 方法的逆序遍历循环中，keys 切片在同一个迭代中可能被添加两次（第 700 行和第 713 行）。这会导致同一个 key 被删除两次。  
**影响**: 逻辑混乱，虽然 del() 是幂等的，但这是逻辑错误  
**修复建议**: 使用 map 去重，或者修正逻辑避免重复添加。

### Bug 6: implmap.Add() 变量遮蔽
**文件**: implmap/implmap.go:41  
**问题描述**: 局部变量 `l` 遮蔽了包级别的锁变量 `l`：
```go
l := len(a)  // 这里的 l 是局部变量，遮蔽了包级别的锁 l
```
**影响**: 代码可读性差，维护困难，容易引入 bug  
**修复建议**: 重命名局部变量：`length := len(a)`

### Bug 7: 不支持 uint 类型转换
**文件**: injector.go:388-434  
**问题描述**: 代码处理 int 和 float 系列的类型转换，但没有处理 uint 系列。如果依赖是 uint 类型，会报错。  
**影响**: 不支持 uint 类型的依赖注入  
**修复建议**: 添加 uint 系列的类型转换支持，与 int 系列类似。

### Bug 8: 结构体标签解析错误处理不一致
**文件**: injector.go:282-303  
**问题描述**: 
- inject 标签解析出错时返回错误
- singleton 和 cannil/nilable 标签解析出错时被静默忽略（使用 _ 接收错误）  
**影响**: 标签语法错误无法被及时发现  
**修复建议**: 统一处理所有标签解析错误。

### Bug 9: Closeable 接口错误被忽略
**文件**: injector.go:718-725  
**问题描述**: 
```go
c, ok := o.Value.(Closeable)
if ok {
    c.Close()  // 错误被忽略
}
```
**影响**: 资源关闭失败无法被检测，可能导致资源泄漏  
**修复建议**: 考虑修改 Closeable 接口返回错误，或至少记录警告日志。

### Bug 10: ReflectRegFields 未检查 nil
**文件**: injector.go:737-757  
**问题描述**: ReflectRegFields 直接调用 Reg()（全局函数），而 Reg() 使用 _g 且未检查 nil。  
**影响**: 如果 InitDefault() 未被调用，会导致 panic  
**修复建议**: 添加 nil 检查或明确文档说明。

### Bug 11: 错误信息不清晰
**文件**: injector.go:210-216  
**问题描述**: 对于非结构体指针类型的 nil 值，错误信息会显示 name=（空）。  
**影响**: 错误信息不清晰，不利于调试  
**修复建议**: 修改错误信息，不显示空的 name 或提供更清晰的描述。

### Bug 12: Logger 接口返回值被忽略
**文件**: define.go:39-44, global.go:51-55  
**问题描述**: Logger 接口的 Error 方法返回 error，但代码中使用时忽略了返回值。  
**影响**: 错误日志记录失败无法被检测  
**修复建议**: 统一处理 Error 的返回值，或修改接口不返回 error。

---

## 代码质量问题 (Code Quality)

### Issue 13: sPrintTree 递归调用注释不足
**文件**: injector.go:605-660  
**问题描述**: sPrintTree 是内部方法，被 SPrintTree（有读锁保护）调用。递归过程中调用了 g.find(tag)，由于外层已经持有读锁，这是安全的，但如果未来修改锁策略，可能会出问题。  
**建议**: 添加注释说明 sPrintTree 必须在持有读锁的情况下调用。

### Issue 14: 锁粒度问题
**文件**: injector.go 多处  
**问题描述**: register() 是递归的，递归调用时锁仍然被持有。这在 Go 的 sync.Mutex 中是允许的（可重入），但在并发场景下可能导致性能问题。  
**建议**: 考虑使用细粒度锁，或者明确文档说明 Graph 的使用限制。

---

## 修复优先级

**P0 (立即修复)**: Bug 1, Bug 2, Bug 3, Bug 4 - 可能导致程序崩溃或竞态条件  
**P1 (建议修复)**: Bug 5, Bug 6, Bug 7, Bug 8, Bug 9 - 影响功能完整性或代码质量  
**P2 (可选修复)**: Bug 10, Bug 11, Bug 12, Issue 13, Issue 14 - 代码健壮性和维护性

---

记录时间: 2026-02-08  
记录者: Code Analysis Tool
