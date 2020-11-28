/*
-------------------------------------------------
   Author :       Zhang Fan
   date：         2020/10/29
   Description :
-------------------------------------------------
*/

package zscheduler

type IObserver interface {
	// 启动中
	Starting()
	// 已启动
	Started()
	// 停止中
	Stopping()
	// 已停止
	Stopped()
	// 已暂停
	Paused()
	// 已恢复
	Resume()

	// 添加任务
	AddTask(task ITask)
	// 移除任务
	RemoveTask(name string)

	// 触发任务
	TriggerTask(task ITask)
	// 尝试添加job失败, 一般为队列已满导致的
	TryAddJobFail(task ITask)
	// job开始
	JobStart(job IJob)
	// job错误, 只有允许重试的task在执行job失败时会通知, 最后一次重试不会通知
	JobErr(job IJob, err error)
	// job结束
	JobEnd(job IJob, executeInfo *ExecuteInfo)
}

var _ IObserver = (*Observer)(nil)

type Observer struct{}

func (*Observer) Starting() {}
func (*Observer) Started()  {}
func (*Observer) Stopping() {}
func (*Observer) Stopped()  {}
func (*Observer) Paused()   {}
func (*Observer) Resume()   {}

func (*Observer) AddTask(task ITask)     {}
func (*Observer) RemoveTask(name string) {}

func (*Observer) TriggerTask(task ITask)                    {}
func (*Observer) TryAddJobFail(task ITask)                  {}
func (*Observer) JobStart(job IJob)                         {}
func (*Observer) JobErr(job IJob, err error)                {}
func (*Observer) JobEnd(job IJob, executeInfo *ExecuteInfo) {}
