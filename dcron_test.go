package dcron

import (
	"fmt"
	dredis "github.com/libi/dcron/driver/redis"
	"github.com/gomodule/redigo/redis"
	"github.com/robfig/cron/v3"
	"log"
	"os"
	"testing"
	"time"
)

type TestJob1 struct {
	Name string
}

func (t TestJob1) Run() {
	fmt.Println("执行 testjob ", t.Name, time.Now().Format("15:04:05"))
}

var testData = make(map[string]struct{})

func Test(t *testing.T) {

	drv, _ := dredis.NewDriver(&dredis.Conf{
		Host: "127.0.0.1",
		Port: 6379,
	}, redis.DialConnectTimeout(time.Second*10))

	go runNode(t, drv)
	go runNode(t, drv)
	go runNode(t, drv)

	//add recover
	dcron2 := NewDcron("server2", drv, cron.WithChain(cron.Recover(cron.DefaultLogger)))

	//panic recover test
	err := dcron2.AddFunc("s2 test1", "* * * * *", func() {
		panic("panic test")
		t.Log("执行 service2 test1 任务,模拟 panic", time.Now().Format("15:04:05"))
	})
	if err != nil {
		t.Fatal("add func error")
	}
	err = dcron2.AddFunc("s2 test2", "* * * * *", func() {
		t.Log("执行 service2 test2 任务", time.Now().Format("15:04:05"))
	})
	if err != nil {
		t.Fatal("add func error")
	}
	err = dcron2.AddFunc("s2 test3", "* * * * *", func() {
		t.Log("执行 service2 test3 任务", time.Now().Format("15:04:05"))
	})
	if err != nil {
		t.Fatal("add func error")
	}
	dcron2.Start()

	// set logger
	logger := log.New(os.Stdout, "[test]", log.LstdFlags)
	// wrap cron recover
	rec := CronOptionChain(cron.Recover(cron.PrintfLogger(logger)))

	// option test
	dcron3 := NewDcronWithOption("server3", drv, rec, WithLogger(logger), WithHashReplicas(10), WithNodeUpdateDuration(time.Second*10))

	//panic recover test
	err = dcron3.AddFunc("s3 test1", "* * * * *", func() {
		t.Log("执行 server3 test1 任务,模拟 panic", time.Now().Format("15:04:05"))
		panic("panic test")
	})
	if err != nil {
		t.Fatal("add func error")
	}
	err = dcron3.AddFunc("s3 test2", "* * * * *", func() {
		t.Log("执行 server3 test2 任务", time.Now().Format("15:04:05"))
	})
	if err != nil {
		t.Fatal("add func error")
	}
	err = dcron3.AddFunc("s3 test3", "* * * * *", func() {
		t.Log("执行 server3 test3 任务", time.Now().Format("15:04:05"))
	})
	if err != nil {
		t.Fatal("add func error")
	}
	dcron3.Start()

	//测试30秒后退出
	time.Sleep(30 * time.Second)
	t.Log("testData", testData)
}

func runNode(t *testing.T, drv *dredis.RedisDriver) {
	dcron := NewDcron("server1", drv)
	//添加多个任务 启动多个节点时 任务会均匀分配给各个节点

	err := dcron.AddFunc("s1 test1", "* * * * *", func() {
		// 同时启动3个节点 但是一个 job 同一时间只会执行一次 通过 map 判重
		key := "s1 test1" + time.Now().Format("15:04:05")
		if _, ok := testData[key]; ok {
			t.Fatal("job have running in other node")
		}
		testData[key] = struct{}{}
	})
	if err != nil {
		t.Fatal("add func error")
	}
	err = dcron.AddFunc("s1 test2", "* * * * *", func() {
		t.Log("执行 service1 test2 任务", time.Now().Format("15:04:05"))
	})
	if err != nil {
		t.Fatal("add func error")
	}

	testJob := TestJob1{"addtestjob"}
	err = dcron.AddJob("addtestjob1", "* * * * *", testJob)
	if err != nil {
		t.Fatal("add func error")
	}

	err = dcron.AddFunc("s1 test3", "* * * * *", func() {
		t.Log("执行 service1 test3 任务", time.Now().Format("15:04:05"))
	})
	if err != nil {
		t.Fatal("add func error")
	}
	dcron.Start()

	//移除测试
	dcron.Remove("s1 test3")
}
