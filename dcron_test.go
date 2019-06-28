package dcron

import (
	"sync"
	"testing"
	"github.com/LibiChai/dcron/driver"
	"fmt"
	"time"
)

func Test(t *testing.T){

	var wg sync.WaitGroup

	wg.Add(1)

	dcron := NewDcronUseRedis("server1",driver.DriverConnOpt{
		Host:"127.0.0.1",
		Port:"6379",
		Password:"",
	})

	dcron2 := NewDcronUseRedis("server2",driver.DriverConnOpt{
		Host:"127.0.0.1",
		Port:"6379",
		Password:"",
	})

	//添加多个任务 启动多个节点时 任务会均匀分配给各个节点
	dcron.AddFunc("s1 test1","*/3 * * * *",func(){
		fmt.Println("执行 service1 test1 任务",time.Now().Format("15:04:05"))
	})
	dcron.AddFunc("s1 test2","*/3 * * * *",func(){
		fmt.Println("执行 service1 test2 任务",time.Now().Format("15:04:05"))
	})
	dcron.AddFunc("s1 test3","*/5 * * * *",func(){
		fmt.Println("执行 service1 test3 任务",time.Now().Format("15:04:05"))
	})
	dcron.Start()

	dcron2.AddFunc("s2 test1","*/3 * * * *",func(){
		fmt.Println("执行 service2 test1 任务",time.Now().Format("15:04:05"))
	})
	dcron2.AddFunc("s2 test2","*/3 * * * *",func(){
		fmt.Println("执行 service2 test2 任务",time.Now().Format("15:04:05"))
	})
	dcron2.AddFunc("s2 test3","*/5 * * * *",func(){
		fmt.Println("执行 service2 test3 任务",time.Now().Format("15:04:05"))
	})
	dcron2.Start()
	//运行多个go test 观察任务分配情况
	wg.Wait()
}
