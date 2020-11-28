/*
-------------------------------------------------
   Author :       Zhang Fan
   date：         2020/11/3
   Description :
-------------------------------------------------
*/

package main

import (
	"fmt"
	"time"

	"github.com/zlyuancn/zscheduler"
)

func main() {
	// 创建一个调度器
	scheduler := zscheduler.NewScheduler()

	// 创建一个任务
	task := zscheduler.NewTask("task1", "@every 1s", func(job zscheduler.IJob) (err error) {
		fmt.Println("执行", time.Now())
		return nil
	})

	// 将任务添加到调度器
	scheduler.AddTask(task)

	// 启动调度器
	scheduler.Start()

	// 等待10秒后结束
	<-time.After(10 * time.Second)
	scheduler.Stop()
}
