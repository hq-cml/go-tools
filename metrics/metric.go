/**
 * go-metrics包是Go领域使用较多的是metrics包，该包是对Java社区依旧十分活跃的Coda Hale’s Metrics library的不完全Go移植
 * （不得不感慨一下：Java的生态还真是强大）。
 *
 * 支持五种Metric类型：
 * 	Gauges ：最简单的度量指标，只有一个简单的返回值，或者叫瞬时状态
 *  Counters：Counter 就是计数器，Counter 只是用 Gauge 封装了 AtomicLong
 *  Meters：Meter度量一系列事件发生的速率(rate)，例如TPS。Meters会统计最近1分钟，5分钟，15分钟，还有全部时间的速率。
 *  Histograms：Histogram统计数据的分布情况。比如最小值，最大值，中间值，还有中位数，75百分位, 90百分位, 95百分位, 98百分位, 99百分位, 和 99.9百分位的值(percentiles)。
 *  Timer其实是 Histogram 和 Meter 的结合， histogram 某部分代码/调用的耗时， meter统计TPS。
 *
 * 参考文档：https://blog.csdn.net/smilejiasmile/article/details/125274894
 */
package metrics

import (
	"fmt"
	"github.com/rcrowley/go-metrics"
	"log"
	"net/http"
	"time"
)

// 瞬时值
func UseGauge() {
	g := metrics.NewGauge()
	tmp := metrics.GetOrRegister("goroutines.now", g)
	_ = tmp
	i := 0
	go func() {
		t := time.NewTicker(1 * time.Second)
		for {
			select {
			case <-t.C:
				g.Update(int64(i))
				//gauge := tmp.(metrics.Gauge)
				//gauge.Update(int64(i))
				i++
				if i >= 10 {
					i = 0
				}
			}
		}
	}()

	time.Sleep(1 * time.Millisecond)

	// 独立goroutine，每秒打印出监控项的值
	// go metrics.Log(metrics.DefaultRegistry, time.Second, log.Default())
	go func() {
		t := time.NewTicker(1 * time.Second)
		for {
			select {
			case <-t.C:
				gauge := tmp.(metrics.Gauge)
				fmt.Println("Value: ", gauge.Value())
			}
		}
	}()

	time.Sleep(100 * time.Second)
}

// 递增值
// 随着curl 'http://127.0.0.1:8080/'，可以看到值会不断递增
func UseCounter() {
	c := metrics.NewCounter()
	metrics.GetOrRegister("total.requests", c)

	go metrics.Log(metrics.DefaultRegistry, time.Second, log.Default())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		c.Inc(1)
	})

	http.ListenAndServe(":8080", nil)
}

// 速率统计
// 参考Timer去理解
func UseMeter() {
	m := metrics.NewMeter()
	metrics.GetOrRegister("rate.requests", m)
	go metrics.Log(metrics.DefaultRegistry, time.Minute, log.Default())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		m.Mark(1)
	})
	http.ListenAndServe(":8080", nil)
}

// 直方图
// 参考Timer去理解
func UseHistogram() {
	//Histogram需要一个采样算法，go-metrics内置了ExpDecaySample采样
	s := metrics.NewExpDecaySample(1028, 0.015)
	h := metrics.NewHistogram(s)
	metrics.GetOrRegister("latency.response", h)
	go metrics.Log(metrics.DefaultRegistry, 20*time.Second, log.Default())

	h.Update(1)
	h.Update(2)
	h.Update(3)
	h.Update(4)
	h.Update(5)
	h.Update(6)
	h.Update(7)
	h.Update(8)
	h.Update(9)
	h.Update(10)

	time.Sleep(100 * time.Second)
}

// Timer其实是 Histogram 和 Meter 的结合，同时得到直方图和速率统计
// 注意：Timer这个东西，通常其实是统计的请求耗时，m.Update(xxx)的参数原型，是一个time.duration
//
//	对于直方图，比较容易理解，就是按照百分比来分布请求耗时
//	对于Meter，则和请求耗时无关，而是通过请求次数来得到QPS
//	          注意这里是QPS！无论是1分钟，5分钟，15分钟，都是QPS
func UseTimer() {
	m := metrics.NewTimer()
	metrics.GetOrRegister("timer.requests", m)
	go metrics.Log(metrics.DefaultRegistry, 5*time.Second, log.Default())

	// case1: 查看请求耗时的到直方图的效果，比较容易理解，就是按照百分比来分布
	//a := 1
	//for i := 0; i < 100; i++ {
	// //m.Update(time.Duration(a) * time.Second)
	// m.Update(50 * time.Millisecond)
	// a++
	//}

	// case2: 查看和请求耗时无关，而是通过请求次数来得到QPS
	//        可以看到随着时间增长，count是不断累积增长的，但是rate是恒定的，因为这里m.Update的速率是恒定的
	for i := 0; i < 100; i++ {
		m.Update(100 * time.Millisecond) // 参数原型，是一个time.duration
		time.Sleep(500 * time.Millisecond)
	}

	time.Sleep(100 * time.Second)
}
