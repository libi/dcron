Dcron
==============
[![Language](https://img.shields.io/badge/Language-Go-blue.svg)](https://golang.org/)
[![Tests](https://github.com/libi/dcron/actions/workflows/test.yml/badge.svg)](https://github.com/libi/dcron/actions/workflows/test.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/libi/dcron#1)](https://goreportcard.com/report/github.com/libi/dcron)

[简体中文](README_CN.md) | [English](README.md)

a lightweight distributed job scheduler  library based on redis or etcd

轻量分布式定时任务库

### 原理

使用 redis/etcd 同步服务节点列表及存活状态，在节点列表内使用一致性hash，选举可执行任务的节点。

### 为什么不直接用分布式锁实现？
通过各个节点在定时任务内抢锁方式实现，需要依赖各个节点系统时间完全一致，当系统时间有误差时可能会导致以下问题：
1. 如果任务的执行时间小于系统时间差，任务仍然会被重复执行（某个节点定时执行完毕释放锁，又被另一个因为系统时间之后到达定时时间的节点取得锁）。
2. 即使有极小的误差，因为某个节点的时间会比其他节点靠前，在抢锁时能第一时间取得锁，所以导致的结果是所有任务都只会被该节点执行，无法均匀分布到多节点。 

### 特性
- 鲁棒性： 任务的节点分配不依赖系统时间，所以各个节点间系统时间有误差也可以确保均匀分布及单节点执行。
- 负载均衡：根据任务数据和节点数据均衡分发任务。
- 无缝扩容：如果任务节点负载过大，直接启动新的服务器后部分任务会自动迁移至新服务实现无缝扩容。
- 故障转移：单个节点故障，10s后会自动将任务自动转移至其他正常节点。
- 任务唯一：同一个服务内同一个任务只会启动单个运行实例，不会重复执行。
- 自定义存储：通过实现driver接口来增加节点数据存储方式。
- 自动恢复：如果进程重启，则**被持久化过的**任务会被自动加载

### 快速开始

1.创建redisDriver实例，指定服务名并初始化dcron。服务名为执行相同任务的单元。
```golang
redisCli := redis.NewClient(&redis.Options{
  Addr: DefaultRedisAddr,
})
drv := driver.NewRedisDriver(redisCli)
dcron := NewDcron("server1", drv)
```
当然，如果你可以自己实现一个自定义的Driver也是可以的，只需要实现[DriverV2](driver/driver.go)接口即可。

2.使用cron语法添加任务，需要指定任务名。任务名作为任务的唯一标识，必须保证唯一。
```golang
dcron.AddFunc("test1","*/3 * * * *",func(){
  fmt.Println("执行 test1 任务",time.Now().Format("15:04:05"))
})
```
3.开始任务。
```golang
// 启动任务可使用 Start() 或者 Run()
// 使用协程异步启动任务
dcron.Start()

// 使用当前协程同步启动任务，会阻塞当前协程后续逻辑执行
dcron.Run()
```

### 使用案例
- [examples](examples/)
- [example_app](https://github.com/dxyinme/dcron_example_app)


### 更多配置

Dcron 项目基于 https://github.com/robfig/cron , 使用 NewDcron 初始化 Dcron 时的第三个参数之后都会原样配置到 cron 。

例如需要配置秒级的 cron 表达式，可以使用

```golang
dcron := NewDcron("server1", drv,cron.WithSeconds())
```

另外还可以通过 ```NewDcronWithOption``` 方法初始化，可以配置日志输出等。
可选配置可以参考：https://github.com/libi/dcron/blob/master/option.go


### 服务名/serviceName

服务名是为了定义相同一组任务，可以理解为任务分配和调度的边界。

多个节点使用同一个服务名会被视为同一任务组，在同一个任务组内的任务会均匀分配至组内各个节点并确保不会重复执行

### Star 历史

[![Star History Chart](https://api.star-history.com/svg?repos=libi/dcron&type=Date)](https://star-history.com/#libi/dcron&Date)