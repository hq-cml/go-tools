package order_map

import (
	"fmt"
)

// OrderedMap 是一个有序映射结构，保持键值对的插入顺序
// 注意：键值对在有序映射中的顺序与插入顺序一致，而不是键的顺序！！！
type OrderedMap struct {
	store  map[interface{}]interface{} // 存储键值对的实际映射
	mapper map[interface{}]*node       // 键到节点的映射，用于快速查找
	// 这是一个空节点，所以它初始化后永远不变，
	// 当链表为空的时候，prev和next均指向自身
	// 当链表非空的时候，prev指向tail，next指向head
	root *node // 双向链表的根节点
}

// NewOrderedMap 创建一个新的有序映射实例
func NewOrderedMap() *OrderedMap {
	om := &OrderedMap{
		store:  make(map[interface{}]interface{}),
		mapper: make(map[interface{}]*node),
		root:   newRootNode(),
	}
	return om
}

// NewOrderedMapWithArgs 使用给定的键值对参数创建一个新的有序映射
// args: 要初始化的键值对数组
func NewOrderedMapWithArgs(args []*KVPair) *OrderedMap {
	om := NewOrderedMap()
	om.update(args)
	return om
}

// update 更新有序映射中的多个键值对
// args: 要更新的键值对数组
func (om *OrderedMap) update(args []*KVPair) {
	for _, pair := range args {
		om.Set(pair.Key, pair.Value)
	}
}

// Set 在有序映射中设置键值对，如果键不存在则同时维护顺序
// key: 要设置的键
// value: 要设置的值
func (om *OrderedMap) Set(key interface{}, value interface{}) {
	if _, ok := om.store[key]; ok == false {
		root := om.root
		last := root.Prev
		last.Next = newNode(last, root, key)
		root.Prev = last.Next
		om.mapper[key] = last.Next
	}
	om.store[key] = value
}

// Get 从有序映射中获取指定键对应的值
func (om *OrderedMap) Get(key interface{}) (interface{}, bool) {
	val, ok := om.store[key]
	return val, ok
}

// Delete 从有序映射中删除指定键及其对应的值
func (om *OrderedMap) Delete(key interface{}) {
	_, ok := om.store[key]
	if ok {
		delete(om.store, key)
	}
	root, rootFound := om.mapper[key]
	if rootFound {
		prev := root.Prev
		next := root.Next
		prev.Next = next
		next.Prev = prev
	}
}

// IterFunc 返回一个函数类型的迭代器，用于遍历有序映射
// 返回值: 函数类型的迭代器，每次调用返回当下键值对键值对和当下是否是root
// 也就是说如果返回了false，则表示已经到了root，迭代结束
// 充分利用了闭包的特性！
func (om *OrderedMap) IterFunc() func() (*KVPair, bool) {
	var curr *node
	root := om.root
	curr = root.Next
	return func() (*KVPair, bool) {
		for curr != root {
			tmp := curr
			curr = curr.Next
			v, _ := om.store[tmp.Value]
			return &KVPair{tmp.Value, v}, true
		}
		return nil, false
	}
}

// RevIterFunc 返回一个反向函数类型的迭代器，用于反向遍历有序映射
// 返回值: 函数类型的反向迭代器，每次调用返回当下键值对键值对和当下是否是root
// 也就是说如果返回了false，则表示已经到了root，迭代结束
func (om *OrderedMap) RevIterFunc() func() (*KVPair, bool) {
	curr := om.root
	// 似乎无必要
	//for {
	//	if curr.Next == nil || curr.Next == curr || curr == om.root {
	//		break
	//	}
	//	curr = curr.Next
	//}

	start := curr
	curr = start.Prev
	return func() (*KVPair, bool) {
		for curr != start {
			tmp := curr
			curr = curr.Prev
			v, _ := om.store[tmp.Value]
			return &KVPair{tmp.Value, v}, true
		}
		return nil, false
	}
}

// String 返回有序映射的字符串表示
// 返回有序映射的字符串格式
func (om *OrderedMap) String() string {
	builder := make([]string, len(om.store))

	var index int = 0
	iter := om.IterFunc()
	for kv, ok := iter(); ok; kv, ok = iter() {
		//val, _ := om.Get(kv.Key)
		builder[index] = fmt.Sprintf("%v:%v, ", kv.Key, kv.Value)
		index++
	}
	return fmt.Sprintf("OrderedMap%v", builder)
}

// UnsafeIter 返回一个不安全的通道迭代器，可能泄漏goroutine（阻塞通道，如果不完整消费，goroutine就泄露了）
// 返回值: 通道类型的键值对迭代器
func (om *OrderedMap) UnsafeIter() <-chan *KVPair {
	keys := make(chan *KVPair)
	go func() {
		defer close(keys)
		var curr *node
		root := om.root
		curr = root.Next
		for curr != root {
			v, _ := om.store[curr.Value]
			keys <- &KVPair{curr.Value, v}
			curr = curr.Next
		}
	}()
	return keys
}

// Iter 返回一个通道迭代器（已弃用）
// 返回值: 通道类型的键值对迭代器
func (om *OrderedMap) Iter() <-chan *KVPair {
	println("Iter() method is deprecated!. Use IterFunc() instead.")
	return om.UnsafeIter()
}

// Len 获取有序映射中键值对的数量
// 返回值: 映射中键值对的数量
func (om *OrderedMap) Len() int {
	return len(om.store)
}
