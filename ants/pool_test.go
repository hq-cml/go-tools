package ants

import (
    "context"
    "log"
    "sync"
    "sync/atomic"
    "testing"
    "time"
)

func Test_goPool_Submit(t *testing.T) {
    var id int32
    pool := New("mypool_nonblock", 3, 1, 10)
    log.Println("Init Pool")
    ctx := context.Background()
    wg := sync.WaitGroup{}
    // 模拟协程数不够，默认阻塞等待
    for i:=0; i<5; i++ {
        wg.Add(1)
        pool.Submit(ctx, func() error {
            defer wg.Done()
            gid := atomic.AddInt32(&id, 1)
            log.Println("Go id:", gid, " Begin Run!")
            time.Sleep(time.Second)
            log.Println("Go id:", gid, " End!")
            return nil
        })
    }
    log.Println("Subbmit Over")
    wg.Wait()
    log.Println("Main Over")
}

func Test_goPool_SubmitNonBlock(t *testing.T) {
    var id int32
    pool := New("mypool_nonblock", 3, 1, 10,
        WithSubmitNonBlock(true),
        WithSubmitRetryIntervalMs(100))
    log.Println("Init Pool")
    ctx := context.Background()
    wg := sync.WaitGroup{}
    // 模拟协程数不够，非阻塞，则会出现循环尝试Submit
    for i:=0; i<5; i++ {
        wg.Add(1)
        pool.Submit(ctx, func() error {
            defer wg.Done()
            gid := atomic.AddInt32(&id, 1)
            log.Println("Go id:", gid, " Begin Run!")
            time.Sleep(time.Second)
            log.Println("Go id:", gid, " End!")
            return nil
        })
    }
    log.Println("Subbmit Over")
    wg.Wait()
    log.Println("Main Over")
}