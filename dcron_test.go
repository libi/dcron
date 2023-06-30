package dcron_test

import (
	"fmt"
	"log"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/gwind55/dcron"
	"github.com/gwind55/dcron/dlog"
	"github.com/gwind55/dcron/driver"
	"github.com/redis/go-redis/v9"
	"github.com/robfig/cron/v3"
	"github.com/stretchr/testify/require"
)

const (
	DefaultRedisAddr = "127.0.0.1:6379"
)

type TestJob1 struct {
	Name string
}

func (t TestJob1) Run() {
	fmt.Println("执行 testjob ", t.Name, time.Now().Format("15:04:05"))
}

var testData = make(map[string]struct{})

func TestMultiNodes(t *testing.T) {
	wg := &sync.WaitGroup{}
	wg.Add(3)

	go runNode(t, wg)
	// 间隔1秒启动测试节点刷新逻辑
	time.Sleep(time.Second)
	go runNode(t, wg)
	time.Sleep(time.Second)
	go runNode(t, wg)

	wg.Wait()
}

func runNode(t *testing.T, wg *sync.WaitGroup) {
	redisCli := redis.NewClient(&redis.Options{
		Addr: DefaultRedisAddr,
	})
	drv := driver.NewRedisDriver(redisCli)
	dcron := dcron.NewDcron("server1", drv)
	//添加多个任务 启动多个节点时 任务会均匀分配给各个节点

	err := dcron.AddFunc("s1 test1", "* * * * *", func() {
		// 同时启动3个节点 但是一个 job 同一时间只会执行一次 通过 map 判重
		key := "s1 test1 : " + time.Now().Format("15:04")
		if _, ok := testData[key]; ok {
			t.Error("job have running in other node")
		}
		testData[key] = struct{}{}
	})
	if err != nil {
		t.Error("add func error")
	}
	err = dcron.AddFunc("s1 test2", "* * * * *", func() {
		t.Log("执行 service1 test2 任务", time.Now().Format("15:04:05"))
	})
	if err != nil {
		t.Error("add func error")
	}

	testJob := TestJob1{"addtestjob"}
	err = dcron.AddJob("addtestjob1", "* * * * *", testJob)
	if err != nil {
		t.Error("add func error")
	}

	err = dcron.AddFunc("s1 test3", "* * * * *", func() {
		t.Log("执行 service1 test3 任务", time.Now().Format("15:04:05"))
	})
	if err != nil {
		t.Error("add func error")
	}
	dcron.Start()

	//移除测试
	dcron.Remove("s1 test3")
	<-time.After(120 * time.Second)
	wg.Done()
	dcron.Stop()
}

func Test_SecondsJob(t *testing.T) {
	redisCli := redis.NewClient(&redis.Options{
		Addr: DefaultRedisAddr,
	})
	drv := driver.NewRedisDriver(redisCli)
	dcr := dcron.NewDcronWithOption(t.Name(), drv, dcron.CronOptionSeconds())
	err := dcr.AddFunc("job1", "*/5 * * * * *", func() {
		t.Log(time.Now())
	})
	if err != nil {
		t.Error(err)
	}
	dcr.Start()
	time.Sleep(15 * time.Second)
	dcr.Stop()
}

func runSecondNode(id string, wg *sync.WaitGroup, runningTime time.Duration, t *testing.T) {
	redisCli := redis.NewClient(&redis.Options{
		Addr: DefaultRedisAddr,
	})
	drv := driver.NewRedisDriver(redisCli)
	dcr := dcron.NewDcronWithOption(t.Name(), drv,
		dcron.CronOptionSeconds(),
		dcron.WithLogger(&dlog.StdLogger{
			Log: log.New(os.Stdout, "["+id+"]", log.LstdFlags),
		}),
		dcron.CronOptionChain(cron.Recover(
			cron.DefaultLogger,
		)),
	)
	var err error
	err = dcr.AddFunc("job1", "*/5 * * * * *", func() {
		t.Log(time.Now())
	})
	require.Nil(t, err)
	err = dcr.AddFunc("job2", "*/8 * * * * *", func() {
		panic("test panic")
	})
	require.Nil(t, err)
	err = dcr.AddFunc("job3", "*/2 * * * * *", func() {
		t.Log("job3:", time.Now())
	})
	require.Nil(t, err)
	dcr.Start()
	<-time.After(runningTime)
	dcr.Stop()
	wg.Done()
}

func Test_SecondJobWithPanicAndMultiNodes(t *testing.T) {
	wg := &sync.WaitGroup{}
	wg.Add(5)
	go runSecondNode("1", wg, 45*time.Second, t)
	go runSecondNode("2", wg, 45*time.Second, t)
	go runSecondNode("3", wg, 45*time.Second, t)
	go runSecondNode("4", wg, 45*time.Second, t)
	go runSecondNode("5", wg, 45*time.Second, t)
	wg.Wait()
}

func Test_SecondJobWithStopAndSwapNode(t *testing.T) {
	wg := &sync.WaitGroup{}
	wg.Add(2)
	go runSecondNode("1", wg, 60*time.Second, t)
	go runSecondNode("2", wg, 20*time.Second, t)
	wg.Wait()
}
