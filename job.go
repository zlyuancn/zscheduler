/*
-------------------------------------------------
   Author :       Zhang Fan
   date：         2020/11/28
   Description :
-------------------------------------------------
*/

package zscheduler

import (
	"time"
)

type IJob interface {
	// 获取task
	Task() ITask
	// 获取元数据
	Meta() interface{}
	// 设置元数据
	SetMeta(meta interface{})
}

type Job struct {
	task ITask
	meta interface{}
}

// 执行信息
type ExecuteInfo struct {
	// 执行开始时间
	ExecuteStartTime time.Time
	// 执行结束时间
	ExecuteEndTime time.Time
	// 是否执行成功
	ExecuteSuccess bool
	// 执行的错误, 如果最后一次执行没有错误那么值为nil
	ExecuteErr error
}

func newJob(task ITask) IJob {
	return &Job{
		task: task,
		meta: nil,
	}
}

func (j *Job) Task() ITask {
	return j.task
}

func (j *Job) Meta() interface{} {
	return j.meta
}
func (j *Job) SetMeta(meta interface{}) {
	j.meta = meta
}
