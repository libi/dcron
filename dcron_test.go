package dcron

import (
	"fmt"
	dredis "github.com/LibiChai/dcron/driver/redis"
	"github.com/gomodule/redigo/redis"
	"github.com/robfig/cron/v3"
	"testing"
	"time"
)

type TestJob1 struct {
	Name string
}

func (t TestJob1) Run() {
	fmt.Println("执行 testjob ", t.Name, time.Now().Format("15:04:05"))
}

func Test(t *testing.T) {

	drv, _ := dredis.NewDriver(&dredis.Conf{
		Host: "127.0.0.1",
		Port: 6379,
	}, redis.DialConnectTimeout(time.Second*10))
	dcron := NewDcron("server1", drv)
	//添加多个任务 启动多个节点时 任务会均匀分配给各个节点

	dcron.AddFunc("s1 test1", "* * * * *", func() {
		fmt.Println("执行 service1 test1 任务", time.Now().Format("15:04:05"))
	})
	dcron.AddFunc("s1 test2", "* * * * *", func() {
		fmt.Println("执行 service1 test2 任务", time.Now().Format("15:04:05"))
	})

	testJob := TestJob1{"addtestjob"}
	dcron.AddJob("addtestjob1", "* * * * *", testJob)

	dcron.AddFunc("s1 test3", "* * * * *", func() {
		fmt.Println("执行 service1 test3 任务", time.Now().Format("15:04:05"))
	})
	dcron.Start()
	// 移除测试
	dcron.Remove("s1 test3")

	//add recover
	dcron2 := NewDcron("server2", drv, cron.WithChain(cron.Recover(cron.DefaultLogger)))

	//panic recover test
	dcron2.AddFunc("s2 test1", "* * * * *", func() {
		panic("panic test")
		fmt.Println("执行 service2 test1 任务,模拟 panic", time.Now().Format("15:04:05"))
	})
	dcron2.AddFunc("s2 test2", "* * * * *", func() {
		fmt.Println("执行 service2 test2 任务", time.Now().Format("15:04:05"))
	})
	dcron2.AddFunc("s2 test3", "* * * * *", func() {
		fmt.Println("执行 service2 test3 任务", time.Now().Format("15:04:05"))
	})
	dcron2.Start()
	//运行多个go test 观察任务分配情况

	//测试120秒后退出
	time.Sleep(120 * time.Second)
}
