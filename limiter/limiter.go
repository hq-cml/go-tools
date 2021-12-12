/*
 * golang官方限流器
 * 限流器是提升服务稳定性的非常重要的组件，可以用来限制请求速率，保护服务，以免服务过载。
 * 限流器的实现方法有很多种，常见的限流算法有固定窗口、滑动窗口、漏桶、令牌桶等。
 *
 * 官方限流器用的是令牌桶方法：
 *     简单来说，令牌桶就是想象有一个固定大小的桶，系统会以恒定速率向桶中放 Token，桶满则暂时不放。
 *     在请求比较的少的时候桶可以先"攒"一些Token，应对突发的流量，如果桶中有剩余 Token 就可以一直取。
 *     如果没有剩余 Token，则需要等到桶中被放置了 Token 才行。
 *
 * 参考：https://studygolang.com/articles/35102?fr=sidebar
 */
package limiter

import (
    "context"
    "golang.org/x/time/rate"
    "log"
    "sync"
    "sync/atomic"
    "time"
)

// 创建并使用一个限速器：
// 每秒钟向桶中放入2个令牌，桶大小10
// 也就是说，最多情况下，一个时刻能允许有10个并发，然后每秒钟2个并发
// 这个例子，从日志的时间可以看到，符合预期：
//    一开始有10个并发，然后每秒钟只允许2个并发
//    并且由于设置了超时时间是3秒，最后的
func NewLimiterWait() {
    var id int32
    cnt := 20
    limiter := rate.NewLimiter(2, 10)
    log.Println("Begin Run!")
    wg := sync.WaitGroup{}
    for i:=0; i<cnt; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            goId := atomic.AddInt32(&id, 1)
            log.Printf("Go [%v] begin run!\n", goId)
            // Wait 方法有一个 context 参数。
            // 可以设置 context 的 Deadline 或者 Timeout，来决定此次 Wait 的最长时间。
            //err := limiter.Wait(context.Background())
            ctx, _ := context.WithTimeout(context.Background(), time.Second * 3)
            err := limiter.Wait(ctx)
            if err != nil {
                log.Printf("Go Running [%v], Error:%v\n", goId, err)
                return
            }
            log.Printf("Go [%v] End!\n", goId)
        }()
    }
    wg.Wait()
    log.Println("Main End!")
}