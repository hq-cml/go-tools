/*
 * 封装一个协程池，底层用的是ants库
 * 注意: ants只是并发度的控制，它不能取代同步机制(如sync.WaitGroup)
 */
package ants

import (
	"context"
	"errors"
	"github.com/panjf2000/ants/v2"
	"log"
	"time"
)

type GoPool interface {
	// Submit 加入任务，任务将异步并发执行，必须确保 task 之间是并发安全的
	Submit(ctx context.Context, task func() error)
}

type conf struct {
	Workers         int `json:"workers"`
	Retries         int `json:"retries"`
	RetryIntervalMs time.Duration `json:"retry_interval_ms"`
	SubmitNonBlock  bool `json:"submit_non_block"`   // 当协程池满了之后，如何处理，默认是阻塞处理，即会一直等待
	SubmitRetryIntervalMs time.Duration `json:"retry_interval_ms"` // 当协程池满了之后，如果是非阻塞，则定期重新尝试submit
}

type goPool struct {
	name string
	pool *ants.Pool
	conf *conf
}

// New 新建协程池
func New(name string, workers, retries, retryIntervalMs int, opts ...Option) GoPool {
	paramOptions := reloadOptions(opts...)
	var antsOpts []ants.Option
	submitRetry := time.Duration(paramOptions.SubmitRetryIntervalMs) * time.Millisecond
	if paramOptions.SubmitNonBlock {
		antsOpts = append(antsOpts, ants.WithNonblocking(true))
		if paramOptions.SubmitRetryIntervalMs == 0 {
			submitRetry = 10 * time.Millisecond
		}
	}
	pool, err := ants.NewPool(workers, antsOpts...)
	if err != nil {
		return nil
	}
	return &goPool{
		pool: pool,
		name: name,
		conf: &conf{
			Workers:         workers,
			Retries:         retries,
			RetryIntervalMs: time.Duration(retryIntervalMs)*time.Millisecond,
			SubmitNonBlock: paramOptions.SubmitNonBlock,
			SubmitRetryIntervalMs: submitRetry,
	}}
}

// Submit 实现 GoPool 接口的 Submit 方法
// 这里submit实现了同步阻塞的提交方式，如果pool没有设置submitNonBlock，则天然阻塞
// 如果设置了submitNonBlock，则这里通过自旋等待的方式，实现了阻塞，留出一个口子可以做一点其他事情
func (p *goPool) Submit(ctx context.Context, task func() error) {
	for {
		err := p.pool.Submit(func() {
			err := tryDo(
				ctx,
				task,
				p.conf.Retries,
				p.conf.RetryIntervalMs)
			if err != nil {
				log.Printf("AsyncTask[%v] execute error:%v", p.name, err)
			}
		})
		// 当设置了submitNonBlock，且协程池满了之后，会出现ErrPoolOverload错误，则sleep等待
		if err != nil && errors.Is(err, ants.ErrPoolOverload) {
			time.Sleep(p.conf.SubmitRetryIntervalMs)
			// TODO 可以做一点其他事情
			//log.Println("Have no enough goroutine. wait a little time")
			continue
		}
		if err != nil {
			log.Printf("AsyncTask[%v] submit error:%v", p.name, err)
		}
		return
	}
}

// 执行提交的任务，如果失败则按失败次数重试
func tryDo(ctx context.Context, task func() error, times int, interval time.Duration) error {
	i := 0
	for  {
		err := task()
		if err == nil{
			return nil
		}
		log.Printf("task execute err:%v", err)
		// ctx = context.Background()
		if i >= times { // 放弃重试
			return err
		}
		i++
		time.Sleep(interval)
	}
}