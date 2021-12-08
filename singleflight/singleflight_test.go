package singleflight

import (
    "sync"
    "testing"
)

//var goCnt = 20000
var goCnt = 20

// 完全不保护，则全部击穿到了db
func Test_getData(t *testing.T) {
    var wg sync.WaitGroup
    wg.Add(goCnt)
    for i:=0; i<goCnt; i++ {
        go func() {
            defer wg.Done()
            data, _ := getData("haha")
            _ = data
        }()
    }
    wg.Wait()
    t.Log("End")
}

// 利用singleflight保护，则发现效果明显好转
func Test_getDataWithSF(t *testing.T) {
    var wg sync.WaitGroup
    wg.Add(goCnt)
    for i:=0; i<goCnt; i++ {
        go func() {
            defer wg.Done()
            data, _ := getDataWithSF("haha")
            _ = data
        }()
    }
    wg.Wait()
    t.Log("End")
}