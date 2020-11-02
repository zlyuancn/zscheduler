/*
-------------------------------------------------
   Author :       Zhang Fan
   date：         2020/10/28
   Description :
-------------------------------------------------
*/

package zscheduler

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/robfig/cron/v3"
)

// 触发器类型
type TriggerType int

const (
	// Cron触发器
	CronTriggerType TriggerType = iota
	// 一次性触发器
	OnceTriggerType
)

func (t TriggerType) String() string {
	switch t {
	case CronTriggerType:
		return "cron"
	case OnceTriggerType:
		return "once"
	}
	return fmt.Sprintf("undefined trigger type: %d", t)
}

type ITrigger interface {
	// 触发器类型
	TriggerType() TriggerType
	// 返回触发器表达式
	Expression() string
	// 返回触发器信息
	TriggerInfo() *TriggerInfo

	// 心跳, 返回是否触发执行
	HeartBeat(t time.Time) (trigger bool)
	// 重置计时器
	ResetClock()
	// 返回下次触发事件
	NextTriggerTime() time.Time
}

type TriggerInfo struct {
	// 触发器类型
	TriggerType TriggerType
	// 表达式
	Expression string
	// 下一次触发时间
	NextTriggerTime time.Time
}

// cron触发器
type CronTrigger struct {
	expression      string
	schedule        cron.Schedule
	nextExecuteTime time.Time
	mx              sync.Mutex
}

// 创建一个cron触发器
func NewCronTrigger(expression string) ITrigger {
	schedule, err := cron.ParseStandard(expression)
	if err != nil {
		panic(fmt.Errorf("expression syntax error, %s", err))
	}

	return &CronTrigger{
		expression:      expression,
		schedule:        schedule,
		nextExecuteTime: schedule.Next(time.Now()),
	}
}

func (c *CronTrigger) TriggerType() TriggerType {
	return CronTriggerType
}
func (c *CronTrigger) Expression() string {
	return c.expression
}
func (c *CronTrigger) TriggerInfo() *TriggerInfo {
	c.mx.Lock()
	nextExecuteTime := c.nextExecuteTime
	c.mx.Unlock()

	return &TriggerInfo{
		TriggerType:     CronTriggerType,
		Expression:      c.expression,
		NextTriggerTime: nextExecuteTime,
	}
}

func (c *CronTrigger) HeartBeat(t time.Time) (trigger bool) {
	c.mx.Lock()

	if t.Unix() < c.nextExecuteTime.Unix() {
		c.mx.Unlock()
		return false
	}

	for t.Unix() >= c.nextExecuteTime.Unix() {
		c.nextExecuteTime = c.schedule.Next(c.nextExecuteTime)
	}

	c.mx.Unlock()

	return true
}
func (c *CronTrigger) ResetClock() {
	c.mx.Lock()
	c.nextExecuteTime = c.schedule.Next(time.Now())
	c.mx.Unlock()
}
func (c *CronTrigger) NextTriggerTime() time.Time {
	c.mx.Lock()
	t := c.nextExecuteTime
	c.mx.Unlock()
	return t
}

// 一次性触发器
type OnceTrigger struct {
	expression  string
	executeTime time.Time
	isRun       int32
}

func NewOnceTrigger(t time.Time) ITrigger {
	o := &OnceTrigger{
		expression:  t.Format(TimeLayout),
		executeTime: t,
	}
	if time.Now().Unix() >= t.Unix() {
		o.isRun = 1
	}
	return o
}

func (o *OnceTrigger) TriggerType() TriggerType {
	return OnceTriggerType
}
func (o *OnceTrigger) Expression() string {
	return o.expression
}
func (o *OnceTrigger) TriggerInfo() *TriggerInfo {
	return &TriggerInfo{
		TriggerType:     OnceTriggerType,
		Expression:      o.expression,
		NextTriggerTime: o.executeTime,
	}
}

func (o *OnceTrigger) HeartBeat(t time.Time) (trigger bool) {
	if atomic.LoadInt32(&o.isRun) == 1 {
		return false
	}

	if t.Unix() >= o.executeTime.Unix() {
		return atomic.CompareAndSwapInt32(&o.isRun, 0, 1)
	}
	return false
}
func (o *OnceTrigger) ResetClock() {
}
func (o *OnceTrigger) NextTriggerTime() time.Time {
	return o.executeTime
}
