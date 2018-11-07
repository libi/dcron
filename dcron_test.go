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

	dcron.AddFunc("test1","*/3 * * * *",func(){
		fmt.Println("执行 test1 任务",time.Now().Format("15:04:05"))
	})
	dcron.Start()

	time.Sleep(time.Second*10000)

}
