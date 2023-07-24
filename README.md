Dcron
==============
[![Language](https://img.shields.io/badge/Language-Go-blue.svg)](https://golang.org/)
[![Tests](https://github.com/libi/dcron/actions/workflows/test.yml/badge.svg)](https://github.com/libi/dcron/actions/workflows/test.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/libi/dcron#1)](https://goreportcard.com/report/github.com/libi/dcron)

[简体中文](README_CN.md) | [English](README.md)

a lightweight distributed job scheduler library based on redis or etcd

### Theory

Use redis or etcd to sync the services list and the state of services. Use consistent-hash
to select the node which can execute the task.

### Why not use distributed-lock?

If use distributed-lock to implement it. I will depends on the system-time of each node. There are some problems when the system-time is not synchronous: 
1. If the task executing time is shorter than the system time, the task will be excuted again. (some node unlock after execution, but the lock will be locked by the other node which reach the execution time)

2. Whatever there is only a little miss in system time, the most fast node will catch the lock in the first time. It will cause a thing that all the task will be executed only by the most fast node.

### Features 
- Robustness: The node assignment of tasks does not depend on the system time, so the system time error between nodes can also ensure uniform distribution and single node execution.
- Load Balance: Distribute tasks evenly according to task and node.
- Seamless capacity extention ：If the load of the task node is too large, some tasks will be automatically migrated to the new service after the new server is started directly to achieve seamless capacity extention. 
- Failover: If a single node fails, the task will be automatically transferred to other normal nodes after 10 seconds. 
- Unique: The same task in the same service will only start a single running instance, and will not be executed repeatedly. 
- Customized storage: add the node storage mode by implementing the driver interface.
- Automatic recovery: if the process restart, the jobs which **have been store** will be recovered into memory.

### Get Started

1. Create redisDriver instance, set the `ServiceName` and initialize `dcron`. The `ServiceName` will defined the same task unit.
```golang
  drv, _ := redis.NewDriver(&redis.Options{
    Host: "127.0.0.1:6379"
  })
  dcron := NewDcron("server1", drv)
```
2. Use cron-language to add task, you should set the `TaskName`, the `TaskName` is the primary-key of each task.
```golang
    dcron.AddFunc("test1","*/3 * * * *",func(){
		fmt.Println("execute test1 task",time.Now().Format("15:04:05"))
	})
```
3. Begin the task
```golang
// you can use Start() or Run() to start the dcron.
// unblocking start.
dcron.Start()

// blocking start.
dcron.Run()
```

### Example
[examples](examples/)


### More configurations.

Dcron is based on https://github.com/robfig/cron, use `NewDcron` to initialize `Dcron`, the arg after the second argv will be passed to `cron`

For example, if you want to set the cron eval in second-level, you can use like that:
```golang
dcron := NewDcron("server1", drv,cron.WithSeconds())
```

Otherwise, you can sue `NewDcronWithOption` to initialize, to set the logger or others. Optional configuration can be referred to: https://github.com/libi/dcron/blob/master/option.go

### ServiceName

The `ServiceName` is used to define the same set of tasks, which can be understood as the boundary of task allocation and scheduling. 

Multiple nodes using the same service name will be considered as the same task group. Tasks in the same task group will be evenly distributed to each node in the group and will not be executed repeatedly.

### Star history

[![Star History Chart](https://api.star-history.com/svg?repos=libi/dcron&type=Date)](https://star-history.com/#libi/dcron&Date)