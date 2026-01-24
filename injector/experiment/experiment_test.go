// 一些常用的反射库方法，进行实验，验证用法
package experiment

import (
	"fmt"
	"os"
	"reflect"
	"testing"
)

type Person struct {
	Name string
	Age  int
}

// 测试 reflect.New
// 功能	为指定类型动态分配内存，并返回指向该内存的指针的 reflect.Value。
// 输入	reflect.Type（通过 reflect.TypeOf(YourType{}) 获得）。
// 输出	reflect.Value，其 Kind() 为 Ptr，指向新创建的零值实例。
// 类比	相当于在堆上执行 &YourType{} 或 new(YourType)，但这一切都是在运行时通过类型信息完成的。
func Test_ReflectNew(t *testing.T) {
	// 1. 获取目标类型的 reflect.Type
	personType := reflect.TypeOf(Person{}) // 注意：这里传递的是值类型，而非指针

	// 2. 使用 reflect.New 创建该类型的新实例指针
	// personValue 是一个 reflect.Value，它持有一个 *Person
	personValue := reflect.New(personType) // 相当于 &Person{}

	// 3. 验证输出结果的种类
	fmt.Println("personValue 的种类 (Kind):", personValue.Kind()) // 输出: ptr
	fmt.Println("personValue 的类型 (Type):", personValue.Type()) // 输出: *main.Person

	// 4. 获取指针所指向的实际对象，并修改其字段
	// Elem() 用于解引用指针，获取底层的 Value
	personElem := personValue.Elem() // 现在 personElem 代表一个 Person 值
	// 使用反射设置字段值（字段名必须大写导出）
	personElem.FieldByName("Name").SetString("Alice")
	personElem.FieldByName("Age").SetInt(25)

	// 5. 将 reflect.Value 转换回接口{}，并进行类型断言以使用
	// Interface() 方法返回 reflect.Value 底层持有的接口值
	personPtr := personValue.Interface().(*Person)
	fmt.Printf("最终结果: %+v\n", *personPtr) // 输出: {Name:Alice Age:25}
}

// 测试 reflect.NewAt
// 与 reflect.New() 关键区别在于：New() 在堆上分配新内存
// 而 reflect.NewAt() 不分配内存，只是将一个已有的内存地址“包装”成指定类型的反射对象，使得能通过反射安全地操作该内存
// 输出:
// reflect.New: 指向新内存的指针的 reflect.Value
// reflect.NewAt: 指向给定地址 p 的指针的 reflect.Value
func Test_ReflectNewAt(t *testing.T) {
	// 1. 获取目标类型的 reflect.Type
	personType := reflect.TypeOf(Person{}) // 注意：这里传递的是值类型，而非指针

	// 2. 使用 reflect.NewAt 包装一个空指针
	// personValue 还原interface{}后的值
	// 此时相当于 var personValue *Person
	personValue := reflect.NewAt(personType, nil).Interface()

	fmt.Println(reflect.TypeOf(personValue))
	fmt.Println(reflect.TypeOf(personValue).Kind())

	// 对比两种输出, 可以发现
	// personValue其实就是等价于var ptr *Person
	fmt.Println("--------------")
	vf := reflect.ValueOf(personValue)
	tf := reflect.TypeOf(personValue)
	fmt.Println(vf)
	fmt.Println(tf)
	fmt.Println(vf.IsValid()) // true，（只有var vf reflect.Value，才会是false，显然此时不符合）
	fmt.Println(vf.IsNil())   // true（personValue==nil，所以ValueOf(personValue).isNil is true）
	fmt.Println("--------------")

	// 可以看到这个和上面一模一样，只是换成了我们熟悉的写法
	var ptr *Person
	vf = reflect.ValueOf(ptr)
	tf = reflect.TypeOf(ptr)
	fmt.Println(vf)
	fmt.Println(tf)
	fmt.Println(vf.IsValid())
	fmt.Println(vf.IsNil())

	// 此时，情况发生了变化，ptr终于不再是nil
	fmt.Println("--------------")
	ptr = &Person{}
	vf = reflect.ValueOf(ptr)
	tf = reflect.TypeOf(ptr)
	fmt.Println(vf)
	fmt.Println(tf)
	fmt.Println(vf.IsValid())
	fmt.Println(vf.IsNil())
}

// 测试 reflect.IsValid
// reflect.Value.IsValid() 用于判断一个 reflect.Value 是否持有一个有效的、可用的 Go 值。
// 怎么样才会出现一个"无效“的go值。主要有以下几种情况：
//  1. reflect.Value{} 零值，也就是这样的写法： var v reflect.Value
//  2. 从 nil 接口 通过 reflect.ValueOf 获取的 Value，也就是这样：reflect.ValueOf(nil).IsValid=false
//     PS: 这里有一个重点要对比的情况，是nil的指针，这两种情况不一样：reflect.ValueOf((*int)(nil)).IsValid=true
//  3. 通过越界访问（如数组索引越界、不存在的结构体字段）获取的 Value
//  4. 对非指针、非接口类型的值调用 Elem() 后得到的结果
//
// 总结：
// 它是一个非常重要的安全检查方法，通常在调用其他反射操作（如 Elem(), Field(), Set()）之前使用，以避免运行时 panic。
// 简单来说：如果 IsValid() 返回 false，那么对这个 reflect.Value 的任何操作（除了 String()）都可能导致 panic。
// IsValid() 是更基础的检查。一个 IsValid() 为 false 的 reflect.Value，甚至没有资格去问它是否为 nil
// 【总结： 对于这个方法的理解，基本上，可以凭直觉】
// 注意这里面最容易混淆的一个地方：
// 对比nil接口和nil指针，他们的reflect.ValueOf是不一样的
func Test_ReflectIsValid(t *testing.T) {
	// ========== 场景1: 零值 reflect.Value ==========
	var zeroV reflect.Value // 声明但未初始化
	fmt.Printf("1. 零值 reflect.Value: IsValid()=%v\n", zeroV.IsValid())
	// 输出: false
	// 警告: zeroV.Type() 或 zeroV.Kind() 会 panic!

	// ========== 场景2: 从 nil接口 获取 Value ==========
	// 这是最常见的需要 IsValid() 检查的情况！
	var nilInterface interface{} = nil
	vFromNil := reflect.ValueOf(nilInterface)
	fmt.Printf("2. 从 nil 接口获取: IsValid()=%v\n", vFromNil.IsValid())
	fmt.Println("2. 相当于reflect.ValueOf(nil).IsValid()：", reflect.ValueOf(nil).IsValid())
	// 输出: false
	// 注意：这里 vFromNil 本身是一个有效的 reflect.Value 对象，
	// 但它表示的是“没有持有任何值的状态”

	// ========== 场景3: 从非 nil 值获取 Value ==========
	num := 42
	vFromNum := reflect.ValueOf(num)
	fmt.Printf("3. 从整数获取: IsValid()=%v, Kind()=%v\n",
		vFromNum.IsValid(), vFromNum.Kind())
	// 输出: true, int

	// ========== 场景4: 通过非法反射操作获取 ==========
	// 4.1 访问不存在的结构体字段
	p := Person{Name: "Alice"}
	vp := reflect.ValueOf(p)
	invalidField := vp.FieldByName("Mail") // Person 没有 Mail 字段
	fmt.Printf("4.1 不存在的字段: IsValid()=%v\n", invalidField.IsValid())
	// 输出: false

	// 4.2 数组/切片索引越界
	arr := [3]int{1, 2, 3}
	va := reflect.ValueOf(arr)
	validElem := va.Index(2) // 有效索引
	// invalidElem := va.Index(5) // 越界索引
	fmt.Printf("4.2 有效索引[2]: IsValid()=%v, Value=%v\n",
		validElem.IsValid(), validElem.Int())
	// fmt.Printf("   无效索引[5]: IsValid()=%v\n", invalidElem.IsValid())
	// 输出:
	// 有效索引[2]: IsValid()=true, Value=3
	// 无效索引[5]: IsValid()=false

	// ========== 场景5: 对非指针类型调用 Elem() ==========
	// 只有指针或接口类型的 Value 才能调用 Elem()
	// vNum := reflect.ValueOf(num)
	// vNumElem := vNum.Elem() // 错误！num 是 int，不是指针
	// fmt.Printf("5. 对非指针调用 Elem(): IsValid()=%v\n", vNumElem.IsValid())
	// 输出: false (调用 Elem() 时不会 panic，但结果是无效的 Value)

	fmt.Printf("5. 对指针调用 Elem()\n")
	// 正确示例：对指针调用 Elem()
	ptr := &num
	vPtr := reflect.ValueOf(ptr)
	vPtrElem := vPtr.Elem() // 正确：解引用指针
	fmt.Printf("   对合法指针调用 Elem(): IsValid()=%v, Value=%v\n",
		vPtrElem.IsValid(), vPtrElem.Int())
	// 输出: true, 42

	// 对 nil 指针调用 Elem()
	var ptr1 *int
	vPtr1 := reflect.ValueOf(ptr1)
	fmt.Printf("   对空指针调用: IsValid()=%v\n", vPtr1.IsValid())
	vPtrElem1 := vPtr1.Elem() // 正确：解引用指针
	fmt.Printf("   对空指针调用 Elem(): IsValid()=%v\n", vPtrElem1.IsValid())
	// 输出: false

	// ========== 场景6: 在 Map 中查找不存在的键 ==========
	m := map[string]int{"a": 1}
	vm := reflect.ValueOf(m)
	mapResult := vm.MapIndex(reflect.ValueOf("b")) // 键 "b" 不存在
	fmt.Printf("6. Map中不存在的键: IsValid()=%v\n", mapResult.IsValid())
	// 输出: false

	// 直观对比nil接口和nil指针
	fmt.Println("7. 直接对比nil指针和nil接口:")
	var i interface{}
	var err error
	fmt.Println("  reflect.ValueOf(var i interface{}).IsValid()=>", reflect.ValueOf(i).IsValid())     // false
	fmt.Println("  reflect.ValueOf(var err error).IsValid()=>", reflect.ValueOf(err).IsValid())       // false
	fmt.Println("  reflect.ValueOf(nil).IsValid()=>", reflect.ValueOf(nil).IsValid())                 // false
	fmt.Println("  reflect.ValueOf((*int)(nil)).IsValid()=>", reflect.ValueOf((*int)(nil)).IsValid()) // true
	var s []int
	fmt.Println("  reflect.ValueOf(var s []int).IsValid()=>", reflect.ValueOf(s).IsValid()) // true
	var mm map[string]int
	fmt.Println("  reflect.ValueOf(var m map[string]int).IsValid()=>", reflect.ValueOf(mm).IsValid()) // true
}

// IsNil() 只能用于 Kind 为 Chan, Func, Interface, Map, Pointer, Slice 的 reflect.Value 类型。
// 如果对不属于这些类型的值（如整数、字符串或结构体）调用 IsNil()，会导致程序 panic。
// safeIsNil ，通过isValid，安全地检查一个接口值是否为 nil(包括接口类型本身的nil和接口持有nil指针的情况)
// 返回：isNil, isValid
func safeIsNil(i interface{}) (bool, bool) {
	v := reflect.ValueOf(i)

	// 第一步：检查 reflect.Value 本身是否代表一个无效的值
	// 这是对零值 reflect.Value 的安全防护，但通常直接传参不会遇到
	if !v.IsValid() {
		return true, false
	}

	// 第二步：检查 Kind，只有特定类型才能调用 IsNil
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		// 只有这些类型可以安全地调用 IsNil()
		return v.IsNil(), true
	default:
		// 对于其他类型（如结构体、int、string等），它们不能与 nil 比较
		// 从概念上讲，它们“不是 nil”
		return false, true
	}
}

// 测试 reflect.ValueOf().IsNil
func Test_ReflectIsNil(t *testing.T) {
	var isNil, isValid bool
	// 示例 1: 指针
	var p *int
	isNil, isValid = safeIsNil(p)
	fmt.Printf("1. 空指针: isNil=%v, isValid=%v\n", isNil, isValid) // isNil=true, isValid=true

	num := 42
	p = &num
	isNil, isValid = safeIsNil(p)
	fmt.Printf("2. 非空指针: isNil=%v, isValid=%v\n", isNil, isValid) // isNil=false, isValid=true

	// 示例 2: 切片
	var s []string
	isNil, isValid = safeIsNil(s)
	fmt.Printf("3. nil 切片: isNil=%v, isValid=%v\n", isNil, isValid) // isNil=true, isValid=true

	s = []string{"a"}
	isNil, isValid = safeIsNil(s)
	fmt.Printf("4. 非nil空切片: isNil=%v, isValid=%v\n", isNil, isValid) // isNil=false, isValid=true

	// 示例 3: 接口 (这是最容易困惑的地方！)
	// 根据上面关于IsValid的介绍，这种情况属于无效值，所以触发的是一个无效
	var err error // error 是一个接口类型
	isNil, isValid = safeIsNil(err)
	fmt.Printf("5. !!!!接口变量自身为 nil: isNil=%v, isValid=%v\n", isNil, isValid) // isNil=true, isValid=false

	var myErr *os.PathError // *PathError 实现了 error 接口
	err = myErr             // 将 nil 指针赋值给接口
	isNil, isValid = safeIsNil(err)
	// 此时 err != nil
	// 首先err是一个字面类型是一个接口，所以它就存在动态值
	// 此时它的动态值是 nil，但接口本身不是“空的”
	fmt.Println(err == nil)                                                      // false
	fmt.Printf("6. 接口本身非nil，但持有 nil 指针: isNil=%v, isValid=%v\n", isNil, isValid) // isNil=true, isValid=true
	// 这就是为什么有时 if err != nil 判断会“失灵”，因为 err 接口变量本身非空，但动态值为 nil

	// 示例 4: 结构体 (不能使用 IsNil)
	var pers Person
	isNil, isValid = safeIsNil(pers)
	fmt.Printf("7. 结构体值: isNil=%v, isValid=%v\n", isNil, isValid) // isNil=false, isValid=true
}

func TestAA(t *testing.T) {
	fmt.Println(reflect.ValueOf(nil).IsValid())
	fmt.Println(reflect.TypeOf(nil))
}
