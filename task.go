/*
-------------------------------------------------
   Author :       Zhang Fan
   date：         2020/10/15
   Description :
-------------------------------------------------
*/

package zscheduler

import (
	"sync"
	"sync/atomic"
	"time"

	jsoniter "github.com/json-iterator/go"
)

const TimeLayout = "2006-01-02 15:04:05"

type Job func() (err error)

type ITask interface {
	// 返回任务名
	Name() string
	// 返回启用状态
	Enable() bool
	// 任务信息
	TaskInfo() *TaskInfo
	// 输出
	String() string

	// 设置启用
	SetEnable(enable ...bool)
	// 心跳, 返回是否触发执行
	HeartBeat(t time.Time) (trigger bool)
	// 立即触发执行, 阻塞等待执行结束
	Trigger(errCallback ErrCallback) *ExecuteInfo
	// 重置计时器
	ResetClock()

	// 修改触发器
	ChangeTrigger(trigger ITrigger)
	// 修改执行器
	ChangeExecutor(executor IExecutor)
}

// 任务信息
type TaskInfo struct {
	// 任务名
	Name string
	// 是否启用
	Enable bool
	// 执行成功数
	ExecutedSuccessNum uint64
	// 执行失败数
	ExecuteFailureNum uint64
	// 最后一次执行信息
	LastExecuteInfo *ExecuteInfo

	// 触发器信息
	TriggerInfo *TriggerInfo
	// 执行器信息
	ExecutorInfo *ExecutorInfo
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

type Task struct {
	name       string
	successNum uint64
	failureNum uint64

	job             Job
	lastExecuteInfo *ExecuteInfo

	trigger  ITrigger
	executor IExecutor

	enable int32
	mx     sync.Mutex
}

type TaskConfig struct {
	Trigger  ITrigger
	Executor IExecutor
	Job      Job
	Enable   bool
}

// 创建一个任务
func NewTask(name string, expression string, job Job, enable ...bool) ITask {
	trigger := NewCronTrigger(expression)
	executor := NewExecutor(0, 0)
	return NewTaskOfConfig(name, TaskConfig{
		Trigger:  trigger,
		Executor: executor,
		Job:      job,
		Enable:   len(enable) == 0 || enable[0],
	})
}

// 根据任务配置创建一个任务
func NewTaskOfConfig(name string, config TaskConfig) ITask {
	t := &Task{
		name:     name,
		trigger:  config.Trigger,
		executor: config.Executor,
		job:      config.Job,
	}
	t.SetEnable(config.Enable)
	return t
}

func (t *Task) Name() string {
	return t.name
}
func (t *Task) Enable() bool {
	return atomic.LoadInt32(&t.enable) == 1
}
func (t *Task) TaskInfo() *TaskInfo {
	t.mx.Lock()
	lastInfo := t.lastExecuteInfo
	trigger := t.trigger
	executor := t.executor
	t.mx.Unlock()

	info := &TaskInfo{
		Name:               t.name,
		Enable:             t.Enable(),
		ExecutedSuccessNum: atomic.LoadUint64(&t.successNum),
		ExecuteFailureNum:  atomic.LoadUint64(&t.failureNum),
		LastExecuteInfo:    nil,

		TriggerInfo:  trigger.TriggerInfo(),
		ExecutorInfo: executor.ExecutorInfo(),
	}

	if lastInfo != nil {
		v := *lastInfo
		info.LastExecuteInfo = &v
	}

	return info
}
func (t *Task) String() string {
	taskInfo := t.TaskInfo()
	triggerInfo := taskInfo.TriggerInfo
	executorInfo := taskInfo.ExecutorInfo
	info := map[string]interface{}{
		"name":                taskInfo.Name,
		"enable":              taskInfo.Enable,
		"execute_success_num": taskInfo.ExecutedSuccessNum,
		"execute_failure_num": taskInfo.ExecuteFailureNum,
		"last_execute_info":   nil,

		"trigger_info": map[string]interface{}{
			"trigger_type":            triggerInfo.TriggerType,
			"expression":              triggerInfo.Expression,
			"next_trigger_time":       triggerInfo.NextTriggerTime.Format(TimeLayout),
			"next_trigger_time_stamp": triggerInfo.NextTriggerTime.Unix(),
		},

		"executor_info": map[string]interface{}{
			"max_sync_execute_count": executorInfo.MaxSyncExecuteCount,
			"sync_execute_count":     executorInfo.SyncExecuteCount,
			"max_retry_count":        executorInfo.MaxRetryCount,
			"retry_interval":         executorInfo.RetryInterval.String(),
		},
	}

	if taskInfo.LastExecuteInfo != nil {
		var errMsg string
		if taskInfo.LastExecuteInfo.ExecuteErr != nil {
			errMsg = taskInfo.LastExecuteInfo.ExecuteErr.Error()
		}
		info["last_execute_info"] = map[string]interface{}{
			"execute_start_time":       taskInfo.LastExecuteInfo.ExecuteStartTime.Format(TimeLayout),
			"execute_start_time_stamp": taskInfo.LastExecuteInfo.ExecuteStartTime.Unix(),
			"execute_end_time":         taskInfo.LastExecuteInfo.ExecuteEndTime.Format(TimeLayout),
			"execute_end_time_stamp":   taskInfo.LastExecuteInfo.ExecuteEndTime.Unix(),
			"execute_success":          taskInfo.LastExecuteInfo.ExecuteSuccess,
			"err_message":              errMsg,
		}
	}

	text, _ := jsoniter.ConfigCompatibleWithStandardLibrary.MarshalToString(info)
	return text
}

func (t *Task) SetEnable(enable ...bool) {
	if len(enable) == 0 || enable[0] {
		atomic.StoreInt32(&t.enable, 1)
		t.ResetClock()
	} else {
		atomic.StoreInt32(&t.enable, 0)
	}
}
func (t *Task) HeartBeat(tt time.Time) bool {
	t.mx.Lock()
	trigger := t.trigger
	t.mx.Unlock()
	if !trigger.HeartBeat(tt) {
		return false
	}

	return t.Enable()
}
func (t *Task) Trigger(errCallback ErrCallback) *ExecuteInfo {
	info := &ExecuteInfo{
		ExecuteStartTime: time.Now(),
	}

	if err := t.execute(errCallback); err != nil {
		info.ExecuteErr = err
		atomic.AddUint64(&t.failureNum, 1)
	} else {
		info.ExecuteSuccess = true
		atomic.AddUint64(&t.successNum, 1)
	}

	info.ExecuteEndTime = time.Now()

	t.mx.Lock()
	t.lastExecuteInfo = info
	t.mx.Unlock()
	return info
}
func (t *Task) ResetClock() {
	t.mx.Lock()
	trigger := t.trigger
	t.mx.Unlock()
	trigger.ResetClock()
}

// 执行
func (t *Task) execute(errCallback ErrCallback) error {
	t.mx.Lock()
	executor := t.executor
	t.mx.Unlock()
	return executor.Do(t.job, errCallback)
}

func (t *Task) ChangeTrigger(trigger ITrigger) {
	t.mx.Lock()
	t.trigger = trigger
	t.mx.Unlock()
}
func (t *Task) ChangeExecutor(executor IExecutor) {
	t.mx.Lock()
	t.executor = executor
	t.mx.Unlock()
}
