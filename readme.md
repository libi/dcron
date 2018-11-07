
## 分布式定时任务
### 原理
    基于redis同步节点数据，模拟服务注册。然后将任务名 根据一致性hash 选举出执行该任务的节点。
### 使用说明

1.指定服务名，使用redis配置初始化dcron。服务名为执行相同任务的单元
```golang
   dcron := NewDcronUseRedis("服务名称",driver.DriverConnOpt{
   		Host:"127.0.0.1",
   		Port:"6379",
   		Password:"",
   })
```
2.使用cron语法添加任务，需要指定任务名。任务名作为任务的唯一标识，必须保证唯一。
```golang
    dcron.AddFunc("test1","*/3 * * * *",func(){
		fmt.Println("执行 test1 任务",time.Now().Format("15:04:05"))
	})
```
3.开始任务。
```golang
dcron.Start()
```

注意：一般定时如果和http服务在一起时不用特殊处理；但如果程序内只有该定时任务，需要阻塞主进程以防止主线程直接退出。
