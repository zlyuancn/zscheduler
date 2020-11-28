/*
-------------------------------------------------
   Author :       Zhang Fan
   date：         2020/10/16
   Description :
-------------------------------------------------
*/

package zscheduler

import (
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	jsoniter "github.com/json-iterator/go"
	"go.uber.org/zap"
)

type IScheduler interface {
	// 运行状态
	RunState() RunState
	// 调度器信息
	SchedulerInfo() *SchedulerInfo
	// 输出
	String() string

	// 启动
	Start()
	// 结束
	Stop()
	// 暂停所有任务
	Pause()
	// 恢复所有任务
	Resume()

	// 添加任务, 如果存在同名任务返回false
	AddTask(task ITask) bool
	// 移除任务
	RemoveTask(name string)
	// 获取任务名列表, 按名称排序
	TaskNames() []string
	// 获取任务列表, 按名称排序
	Tasks() []ITask
	// 获取任务, 如果不存在返回nil
	GetTask(name string) ITask
}

// 运行状态
type RunState int32

const (
	// 已停止
	StoppedState RunState = iota
	// 启动中
	StartingState
	// 暂停
	PausedState
	// 恢复中
	ResumingState
	// 已启动
	StartedState
	// 停止中
	StoppingState
)

func (r RunState) String() string {
	switch r {
	case StoppedState:
		return "stopped"
	case StartingState:
		return "starting"
	case PausedState:
		return "paused"
	case ResumingState:
		return "resuming"
	case StartedState:
		return "started"
	case StoppingState:
		return "stopping"
	}
	return fmt.Sprintf("undefined state: %d", r)
}

// 调度器信息
type SchedulerInfo struct {
	// 运行状态
	RunState RunState
	// 任务数
	TaskNum int
	// 执行成功数
	ExecutedSuccessNum uint64
	// 执行失败数
	ExecuteFailureNum uint64
}

type Scheduler struct {
	*Options
	tasks map[string]ITask

	runState  RunState
	closeChan chan struct{}

	successNum uint64
	failureNum uint64

	pause RunState

	mx sync.Mutex
}

func NewScheduler(opts ...Option) IScheduler {
	m := &Scheduler{
		Options:   newOptions(),
		tasks:     make(map[string]ITask),
		runState:  StoppedState,
		closeChan: make(chan struct{}),
		pause:     0,
	}

	for _, o := range opts {
		o(m.Options)
	}
	if m.gpool != nil {
		m.gpool.Start()
	}
	return m
}

func (m *Scheduler) RunState() RunState {
	state := RunState(atomic.LoadInt32((*int32)(&m.runState)))
	if state == StartedState && m.isPause() {
		state = PausedState
	}
	return state
}
func (m *Scheduler) SchedulerInfo() *SchedulerInfo {
	m.mx.Lock()
	taskNum := len(m.tasks)
	m.mx.Unlock()

	info := &SchedulerInfo{
		RunState:           m.RunState(),
		TaskNum:            taskNum,
		ExecutedSuccessNum: atomic.LoadUint64(&m.successNum),
		ExecuteFailureNum:  atomic.LoadUint64(&m.failureNum),
	}
	return info
}
func (m *Scheduler) String() string {
	schedulerInfo := m.SchedulerInfo()
	info := map[string]interface{}{
		"run_state":           schedulerInfo.RunState,
		"run_state_text":      schedulerInfo.RunState.String(),
		"task_num":            schedulerInfo.TaskNum,
		"execute_success_num": schedulerInfo.ExecutedSuccessNum,
		"execute_failure_num": schedulerInfo.ExecuteFailureNum,
	}
	text, _ := jsoniter.ConfigCompatibleWithStandardLibrary.MarshalToString(info)
	return text
}

func (m *Scheduler) Start() {
	if !atomic.CompareAndSwapInt32((*int32)(&m.runState), int32(StoppedState), int32(StartingState)) {
		return
	}
	if m.log != nil {
		m.log.Info("正在启动调度器")
	}
	m.notifier.Starting()
	atomic.StoreInt32((*int32)(&m.pause), int32(ResumingState))

	m.start()

	atomic.StoreInt32((*int32)(&m.pause), int32(StartedState))
	atomic.StoreInt32((*int32)(&m.runState), int32(StartedState))
	if m.log != nil {
		m.log.Info("已启动调度器")
	}
	m.notifier.Started()
}
func (m *Scheduler) Stop() {
	if !atomic.CompareAndSwapInt32((*int32)(&m.runState), int32(StartedState), int32(StoppingState)) {
		return
	}
	if m.log != nil {
		m.log.Info("正在关闭调度器")
	}
	m.notifier.Stopping()
	m.closeChan <- struct{}{}
	<-m.closeChan

	atomic.StoreInt32((*int32)(&m.runState), int32(StoppedState))
	if m.log != nil {
		m.log.Info("已关闭调度器")
	}
	m.notifier.Stopped()
}
func (m *Scheduler) Pause() {
	if m.RunState() != StartedState {
		return
	}

	if atomic.CompareAndSwapInt32((*int32)(&m.pause), int32(StartedState), int32(PausedState)) {
		if m.log != nil {
			m.log.Info("暂停调度器")
		}
		m.notifier.Paused()
	}
}
func (m *Scheduler) Resume() {
	if m.RunState() != StartedState {
		return
	}

	if atomic.CompareAndSwapInt32((*int32)(&m.pause), int32(PausedState), int32(ResumingState)) {
		m.resetClock()

		// 设为启动状态(恢复完成)
		// 这里使用cas是为了防止这个时候用户调用了Stop()+Start()后状态被更改
		if atomic.CompareAndSwapInt32((*int32)(&m.pause), int32(ResumingState), int32(StartedState)) {
			if m.log != nil {
				m.log.Info("恢复调度器")
			}
			m.notifier.Resume()
		}
	}
}

func (m *Scheduler) AddTask(task ITask) bool {
	m.mx.Lock()
	if _, ok := m.tasks[task.Name()]; ok {
		m.mx.Unlock()
		return false
	}

	m.tasks[task.Name()] = task
	task.ResetClock()
	m.mx.Unlock()
	m.notifier.AddTask(task)
	return true
}
func (m *Scheduler) RemoveTask(name string) {
	m.mx.Lock()
	if _, ok := m.tasks[name]; !ok {
		m.mx.Unlock()
		return
	}

	delete(m.tasks, name)
	m.mx.Unlock()
	m.notifier.RemoveTask(name)
}
func (m *Scheduler) TaskNames() []string {
	m.mx.Lock()
	names := make([]string, 0, len(m.tasks))
	for name := range m.tasks {
		names = append(names, name)
	}
	m.mx.Unlock()

	sort.Strings(names)
	return names
}
func (m *Scheduler) Tasks() []ITask {
	m.mx.Lock()
	tasks := make([]ITask, 0, len(m.tasks))
	for _, task := range m.tasks {
		tasks = append(tasks, task)
	}
	m.mx.Unlock()

	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].Name() < tasks[j].Name()
	})
	return tasks
}
func (m *Scheduler) GetTask(name string) ITask {
	m.mx.Lock()
	task := m.tasks[name]
	m.mx.Unlock()
	return task
}

// 开始
func (m *Scheduler) start() {
	m.resetClock()

	go func() {
		timer := time.NewTicker(time.Second)
		for {
			select {
			case t := <-timer.C:
				if !m.isPause() {
					go m.heartBeat(t)
				}
			case <-m.closeChan:
				timer.Stop()
				m.closeChan <- struct{}{}
				return
			}
		}
	}()
}

// 是否暂停中
func (m *Scheduler) isPause() bool {
	return atomic.LoadInt32((*int32)(&m.pause)) != int32(StartedState) // 只要不是运行时都是暂停状态
}

// 心跳
func (m *Scheduler) heartBeat(t time.Time) {
	m.mx.Lock()
	defer m.mx.Unlock()

	for _, task := range m.tasks {
		if !task.Enable() {
			continue
		}

		if task.HeartBeat(t) {
			m.triggerTask(task)
		}
	}
}

// 触发一个任务
func (m *Scheduler) triggerTask(t ITask) {
	m.notifier.TriggerTask(t)
	if m.gpool == nil {
		go m.execute(t)
	} else {
		add := m.gpool.TryAddJob(func() {
			m.execute(t)
		})
		if !add {
			if m.log != nil {
				m.log.Warn("任务生成失败, 因为队列已满", zap.String("name", t.Name()))
			}
			m.notifier.TryAddJobFail(t)
		}
	}
}

// 执行一个任务
func (m *Scheduler) execute(t ITask) {
	if !t.Enable() {
		return
	}

	if m.log != nil {
		m.log.Info("开始执行任务", zap.String("name", t.Name()))
	}
	m.notifier.JobStart(t)
	result := t.Trigger(func(err error) {
		if m.log != nil {
			m.log.Warn("任务执行失败, 即将重试", zap.String("name", t.Name()), zap.Error(err))
		}
		m.notifier.JobErr(t, err)
	})
	if result.ExecuteSuccess {
		atomic.AddUint64(&m.successNum, 1)
		if m.log != nil {
			m.log.Info("任务执行成功", zap.String("name", t.Name()))
		}
	} else {
		atomic.AddUint64(&m.failureNum, 1)
		if m.log != nil {
			m.log.Warn("任务执行失败", zap.String("name", t.Name()), zap.Error(result.ExecuteErr))
		}
	}
	m.notifier.JobEnd(t, result)
}

// 重置计时器
func (m *Scheduler) resetClock() {
	m.mx.Lock()
	for _, t := range m.tasks {
		t.ResetClock()
	}
	m.mx.Unlock()
}
