/*
 * 一个线程安全的类型映射注册表。主要功能：
 * 在公共内存中维护一个名称=>结构体指针(reflect.Type)列表的映射关系(
 * golang的包是全局性的，所以多次import仍然是同一套内存空间)
 * 使用读写锁保证并发安全，内部维护字符串到类型数组的映射关系
 */
package implmap

import (
	"fmt"
	"reflect"
	"sync"
)

// 全局空间
var (
	m = make(map[string][]reflect.Type)
	l = &sync.RWMutex{}
)

// isStructPtr函数：验证类型是否为结构体指针
func isStructPtr(t reflect.Type) bool {
	return t != nil &&
		t.Kind() == reflect.Ptr &&
		t.Elem().Kind() == reflect.Struct
}

// Add函数：向指定名称注册结构体指针类型，支持同名多个类型注册（将以列表形式保存）
// accept struct pointer only
func Add(n string, t reflect.Type) {
	if t == nil || n == "" || !isStructPtr(t) {
		return
	}
	l.Lock()
	defer l.Unlock()
	a, ok := m[n]
	if !ok || a == nil {
		a = []reflect.Type{}
	}

	l := len(a)
	if l > 0 {
		fmt.Println(fmt.Sprintf("implmap append new type(%v) impl to name(%v) at index(%v), old array=%v", t, n, l, a))
	}

	a = append(a, t)
	m[n] = a
}

// Get函数：根据名称获取对应的所有类型数组
func Get(n string) []reflect.Type {
	if n == "" {
		return []reflect.Type{}
	}
	l.RLock()
	defer l.RUnlock()
	types := m[n]
	if types == nil {
		return []reflect.Type{}
	}
	ret := []reflect.Type{}
	for _, t := range types {
		if t == nil {
			continue
		}
		ret = append(ret, t)
	}
	return ret
}

// * GetAll函数：获取完整的类型映射副本
func GetAll() map[string][]reflect.Type {
	l.RLock()
	defer l.RUnlock()
	mCopy := make(map[string][]reflect.Type)
	for key := range m {
		ret := []reflect.Type{}
		for _, t := range m[key] {
			if t == nil {
				continue
			}
			ret = append(ret, t)
			mCopy[key] = ret
		}
	}
	return mCopy
}
