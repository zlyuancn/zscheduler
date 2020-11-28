/*
-------------------------------------------------
   Author :       Zhang Fan
   date：         2020/10/28
   Description :
-------------------------------------------------
*/

package zscheduler

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

var OutOfMaxSyncExecuteCount = errors.New("超出最大同步执行数")

type ErrCallback func(job IJob, err error)

type IExecutor interface {
	// 执行
	Do(job IJob, errCallback ErrCallback) error
	// 等待任务执行完毕
	Wait()
	// 返回是否正在执行任务
	IsRunning() bool
	// 执行器信息
	ExecutorInfo() *ExecutorInfo
}

type ExecutorInfo struct {
	MaxSyncExecuteCount int64         // 最大同步执行数
	SyncExecuteCount    int64         // 同步执行数
	MaxRetryCount       int64         // 最大重试数
	RetryInterval       time.Duration // 重试间隔
}

type Executor struct {
	maxSyncExecuteCount int64         // 最大同步执行数
	syncExecuteCount    int64         // 同步执行数
	maxRetryCount       int64         // 重试次数
	retryInterval       time.Duration // 重试间隔
	mx                  sync.Mutex
	wg                  sync.WaitGroup
}

// 创建一个执行器, 任务失败会重试
//
// maxRetryCount: 任务失败重试次数
// retryInterval: 失败重试间隔时间
// maxSyncExecuteCount: 最大同步执行任务数, 如果为0则不限制
func NewExecutor(retryCount int64, retryInterval time.Duration, maxSyncExecuteCount ...int64) IExecutor {
	count := int64(0)
	if len(maxSyncExecuteCount) > 0 {
		count = maxSyncExecuteCount[0]
	}
	return &Executor{
		maxSyncExecuteCount: count,
		syncExecuteCount:    0,
		maxRetryCount:       retryCount,
		retryInterval:       retryInterval,
	}
}

func (w *Executor) ExecutorInfo() *ExecutorInfo {
	w.mx.Lock()
	syncExecuteCount := w.syncExecuteCount
	w.mx.Unlock()

	return &ExecutorInfo{
		MaxSyncExecuteCount: w.maxSyncExecuteCount,
		SyncExecuteCount:    syncExecuteCount,
		MaxRetryCount:       w.maxRetryCount,
		RetryInterval:       w.retryInterval,
	}
}

// 执行, 如果已经达到最大同步执行任务数则会忽略这个任务
func (w *Executor) Do(job IJob, errCallback ErrCallback) error {
	w.mx.Lock()
	if w.maxSyncExecuteCount > 0 && w.syncExecuteCount >= w.maxSyncExecuteCount {
		w.mx.Unlock()
		return OutOfMaxSyncExecuteCount
	}

	w.wg.Add(1)
	w.syncExecuteCount++
	w.mx.Unlock()

	handler := w.wrapExecute(job.Task().Handler())
	err := w.doRetry(handler, job, w.retryInterval, w.maxRetryCount, errCallback)

	w.mx.Lock()
	w.syncExecuteCount--
	w.wg.Done()
	w.mx.Unlock()

	return err
}

// 等待所有任务执行完毕
func (w *Executor) Wait() {
	w.wg.Wait()
}

func (w *Executor) IsRunning() bool {
	w.mx.Lock()
	run := w.syncExecuteCount > 0
	w.mx.Unlock()
	return run
}

// 包装执行, 拦截panic
func (w *Executor) wrapExecute(handler Handler) Handler {
	return func(job IJob) (err error) {
		defer func() {
			e := recover()
			switch v := e.(type) {
			case nil:
			case error:
				err = v
			case string:
				err = errors.New(v)
			default:
				err = errors.New(fmt.Sprint(err))
			}
		}()

		err = handler(job)
		return
	}
}

// 执行一个函数
func (w *Executor) doRetry(handler Handler, job IJob, interval time.Duration, retryCount int64, errCallback ErrCallback) (err error) {
	for {
		if err = handler(job); err == nil || retryCount == 0 {
			// 这里不需要错误回调
			return
		}

		retryCount--

		if errCallback != nil {
			errCallback(job, err)
		}

		if interval > 0 {
			time.Sleep(interval)
		}
	}
}
