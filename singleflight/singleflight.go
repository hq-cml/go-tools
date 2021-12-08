/*
 * 并发控制，防击穿神器：singleflight
 *
 * 缓存击穿：
 * 缓存在某个时间点过期的时候，恰好在这个时间点对这个Key有大量的并发请求过来，这些请求发现缓存
 * 过期一般都会从后端DB加载数据并回设到缓存，这个时候大并发的请求可能会瞬间把后端DB压垮。
 *
 * 应对措施：
 * 一般这种情况，都是利用一个分布式锁，比如用redis的setnx简单实现一个，不过还是有点小复杂。
 * 如果线上实例不会太多，那么可以考虑直接在本地搞一个锁，直接在本地做一下防击穿（如果分布式实例太多，仍然有风险）
 * 本地并发控制，如果不自己搞锁，那么用singleflight包也可以搞
 * 整个包的核心代码不到100行，充分利用到了map和WaitGroup的特性。
 */
package singleflight

import (
    "errors"
    sf "golang.org/x/sync/singleflight"
    "log"
    "sync/atomic"
)

var term uint32
var errNoExist = errors.New("not exist in cache")
var gsfGrp sf.Group

// 模拟数据的获取
// 通常都是先读取缓存，然后如果缓存不存在，则读db
// 因为完全没有保护（且getDataCache必然失败），所以100%击穿db
func getData(key string) (string, error) {
    data, err := getDataFromCache(key)
    if err == nil {
        return data, nil
    }

    // 缓存不命中，从db获取
    if err == errNoExist {
        data, err = getDataFromDB(key)
        if err != nil {
            return "", errors.New("Never happen here!")
        }

        // TODO 种缓存
        return data, nil
    }
    return "", errors.New("Never happen here!")
}

// 从cache获取，这里模拟无论如何返回未命中错误
func getDataFromCache(key string) (string, error) {
    return "", errNoExist
}

// 模拟从db获取数据
func getDataFromDB(key string) (string, error) {
    log.Printf("Get %s from database(%d)\n", key, atomic.AddUint32(&term, 1))
    return "data", nil
}

// 模拟数据的获取
// 利用singleflight进行并发保护
// 注意：sf并不是说100%一定只有一次，主要是看是否是同一时刻
//     通过返回值shared可以区分出是否完全是一次（Do方法的注释的解释）
func getDataWithSF(key string) (string, error) {
    data, err := getDataFromCache(key)
    if err == nil {
        return data, nil
    }

    // 缓存不命中，从db获取
    if err == errNoExist {
        v, err , shared := gsfGrp.Do(key, func()(interface{}, error) {
            return getDataFromDB(key)
        })
        if err != nil {
            return "", errors.New("Never happen here!")
        }
        if !shared {
            // shared==false，说明发生了另一次调用
            log.Println("Call another time")
        }
        data = v.(string)
        // TODO 种缓存
        return data, nil
    }
    return "", errors.New("Never happen here!")
}
