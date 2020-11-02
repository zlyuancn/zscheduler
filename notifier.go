/*
-------------------------------------------------
   Author :       Zhang Fan
   date：         2020/10/29
   Description :
-------------------------------------------------
*/

package zscheduler

import (
	"sync"
)

type INotifier interface {
	AddObserver(o IObserver)
	IObserver
}

var _ INotifier = (*Notifier)(nil)

type Notifier struct {
	observers []IObserver
	mx        sync.Mutex
}

func newNotifier() INotifier {
	return &Notifier{}
}

func (n *Notifier) AddObserver(o IObserver) {
	n.mx.Lock()
	n.observers = append(n.observers, o)
	n.mx.Unlock()
}

func (n *Notifier) Starting() {
	n.eachObservers(func(o IObserver) {
		o.Starting()
	})
}
func (n *Notifier) Started() {
	n.eachObservers(func(o IObserver) {
		o.Started()
	})
}
func (n *Notifier) Stopping() {
	n.eachObservers(func(o IObserver) {
		o.Stopping()
	})
}
func (n *Notifier) Stopped() {
	n.eachObservers(func(o IObserver) {
		o.Stopped()
	})
}
func (n *Notifier) Paused() {
	n.eachObservers(func(o IObserver) {
		o.Paused()
	})
}
func (n *Notifier) Resume() {
	n.eachObservers(func(o IObserver) {
		o.Resume()
	})
}

func (n *Notifier) AddTask(task ITask) {
	n.eachObservers(func(o IObserver) {
		o.AddTask(task)
	})
}
func (n *Notifier) RemoveTask(name string) {
	n.eachObservers(func(o IObserver) {
		o.RemoveTask(name)
	})
}

func (n *Notifier) TriggerTask(task ITask) {
	n.eachObservers(func(o IObserver) {
		o.TriggerTask(task)
	})
}
func (n *Notifier) JobStart(task ITask) {
	n.eachObservers(func(o IObserver) {
		o.JobStart(task)
	})
}
func (n *Notifier) JobEnd(task ITask, result *ExecuteInfo) {
	n.eachObservers(func(o IObserver) {
		o.JobEnd(task, result)
	})
}

func (n *Notifier) eachObservers(fn func(o IObserver)) {
	n.mx.Lock()
	defer n.mx.Unlock()
	for _, o := range n.observers {
		fn(o)
	}
}
