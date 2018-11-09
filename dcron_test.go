package dcron

import (
	"testing"
	"github.com/LibiChai/dcron/driver"
	"fmt"
	"time"
)

func Test(t *testing.T){

	dcron := NewDcronUseRedis("servername",driver.DriverConnOpt{
		Host:"127.0.0.1",
		Port:"6379",
		Password:"",
	})

	//添加多个任务 使用同一个服务器名 启动多个节点时 任务会均匀分配给各个节点
	dcron.AddFunc("test1","*/3 * * * *",func(){
		fmt.Println("执行 test1 任务",time.Now().Format("15:04:05"))
	})
	dcron.AddFunc("test2","*/3 * * * *",func(){
		fmt.Println("执行 test2 任务",time.Now().Format("15:04:05"))
	})
	dcron.AddFunc("test3","*/5 * * * *",func(){
		fmt.Println("执行 test4 任务",time.Now().Format("15:04:05"))
	})
	dcron.Start()

	time.Sleep(time.Second*10000)

}
