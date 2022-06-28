dcron
==============
[![Language](https://img.shields.io/badge/Language-Go-blue.svg)](https://golang.org/)
[![Tests](https://github.com/libi/dcron/actions/workflows/test.yml/badge.svg)](https://github.com/libi/dcron/actions/workflows/test.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/libi/dcron#1)](https://goreportcard.com/report/github.com/libi/dcron)

分布式定时任务库

### 原理

基于redis同步节点数据，模拟服务注册。然后将任务名 根据一致性hash 选举出执行该任务的节点。

### 流程图

![dcron流程图](dcron.png)

### 特性

- 负载均衡：根据任务数据和节点数据均衡分发任务。
- 无缝扩容：如果任务节点负载过大，直接启动新的服务器后部分任务会自动迁移至新服务实现无缝扩容。
- 故障转移：单个节点故障，10s后会自动将任务自动转移至其他正常节点。
- 任务唯一：同一个服务内同一个任务只会启动单个运行实例，不会重复执行。
- 自定义存储：通过实现driver接口来增加节点数据存储方式。

### 快速开始

1.创建redisDriver实例，指定服务名并初始化dcron。服务名为执行相同任务的单元。
```golang
  drv, _ := redis.NewDriver(&redis.Conf{
  		Host: "127.0.0.1",
  		Port: 6379,
  })
  dcron := NewDcron("server1", drv)
```
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

### 更多配置

Dcron 项目基于 https://github.com/robfig/cron , 使用 NewDcron 初始化 Dcron 时的第三个参数之后都会原样配置到 cron 。

例如需要配置秒级的 cron 表达式，可以使用

```golang
dcron := NewDcron("server1", drv,cron.WithSeconds())
```

另外还可以通过 ```NewDcronWithOption``` 方法初始化，可以配置日志输出等。
可选配置可以参考：https://github.com/libi/dcron/blob/master/option.go


### 关于服务名

服务名只是为了定义相同一组任务，节点在启动时会产生一个uuid，然后绑定到这个服务内，不会存在多个节点使用同一个服务明出现冲突的问题。

比如有个服务叫【课堂服务】里面包含了 【上课】【下课】 等各类定时任务，那么就可以有n个不同的服务节点（可以在同一台或者不同机器上），服务都叫课堂服务。

