
# 朴实无华的调度器

---

[toc]

---

# 功能列表

+ task

- [x] cron定时器
- [x] 一次性定时器
- [x] 重试
- [x] 作业panic拦截
- [x] 并发执行控制(线程限制)
- [x] 详细的执行信息
- [x] 详细的task信息
- [x] 可暂停恢复
- [x] 可修改触发器和执行器(不需要删除这个任务后重新创建任务了)
- [x] 序列化

+ scheduler

- [x] 全局并发执行控制(线程限制)
- [x] 任务查询
- [x] 详细的scheduler信息
- [x] 全局可暂停恢复
- [x] 通告员和观察者
- [x] 详细的日志记录

+ 其他

> 所有组件提供接口, 允许用户自由扩展功能

- [x] 执行器
- [x] 触发器
- [x] 任务
- [x] 调度器
- [x] 通告员
- [x] 观察者

# 获得
`go get -u github.com/zlyuancn/zscheduler`

# 概念说明

## 作业内容(job)
> 描述一个任务

## 执行器(executor)
> 执行器控制job如何去执行, 每一个task有一个自己的执行器, 他可以控制job执行并发限制, 它还包装了job拦截panic导致的错误

## 触发器(trigger)
> 用于决定task在什么时候生成job, 每一个task有一个自己的触发器

## 任务(task)
> 任务实体, 用于将触发器和执行器以及作业内容联系起来, 表示一个完整的工作内容

## 调度器(scheduler)
> 通常一个应用只有一个调度器, 用于管理多个task, 它为触发器提供动力

## 观察者(observer)
> 用于监测调度器的启停以及task的触发执行和执行结果等


# 使用说明

```go
// 创建一个调度器
scheduler := zscheduler.NewScheduler()

// 创建一个任务
task := zscheduler.NewTask("task1", "@every 1s", func() (err error) {
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
```

# 教程

+ [1.简单示例](example/e1.simple/main.go)
+ 待续...
