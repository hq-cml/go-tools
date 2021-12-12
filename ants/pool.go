/*
 * 自己封装一个协程池
 */
package ants

import (
	"context"
	"github.com/panjf2000/ants/v2"
	"time"
	"errors"
)

type GoPool interface {
	// Submit 加入任务，任务将异步并发执行，必须确保 task 之间是并发安全的
	Submit(ctx context.Context, task func() error)
}

type options struct {
	Workers         int `json:"workers"`
	Retries         int `json:"retries"`
	RetryIntervalMs int `json:"retry_interval_ms"`
}

type goPool struct {
	name    string
	pool    *ants.Pool
	options *options
}

// New 构造函数
func New(name string, workers, retries, retryInterval int) GoPool {
	pool, err := ants.NewPool(workers)
	if err != nil {
		return nil
	}
	return &goPool{
		pool: pool,
		name: name,
		options: &options{
			Workers:         workers,
			Retries:         retries,
			RetryIntervalMs: retryInterval,
	}}
}

// Submit 实现 API 接口的 Submit 方法
func (a *goPool) Submit(ctx context.Context, task func() error) {
	for {
		err := a.pool.Submit(func() {
			err := tryDo(
				ctx,
				task,
				a.options.Retries,
				time.Duration(a.options.RetryIntervalMs)*time.Millisecond)
			if err != nil {
				//log.Errorf("AsyncTask[%v] execute error:%v", a.name, err)
			}
		})
		if err != nil {
			//log.Errorf("AsyncTask[%v] submit error:%v", a.name, err)
		}
		if !errors.Is(err, ants.ErrPoolOverload) {
			return
		}
		time.Sleep(time.Duration(a.options.RetryIntervalMs) * time.Millisecond)
	}
}

func tryDo(ctx context.Context, task func() error, times int, interval time.Duration) error {
	i := 0
	for err := task(); err != nil; i++ {
		//log.Errorf("task execute err:%v", err)
		ctx = context.Background()
		if i >= times { // 放弃重试
			return err
		}
		time.Sleep(interval)
	}
	return nil
}