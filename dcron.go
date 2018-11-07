package dcron

import (
	"sync"
	"github.com/robfig/cron"
	"github.com/LibiChai/dcron/driver"
	"fmt"
)



type Dcron struct {
	jobs     map[string]*JobWarpper
	mu       sync.RWMutex
	cr       *cron.Cron
	ServerName string
	nodePool *NodePool
}

func NewDcronUseRedis(serverName string,dataSourceOption driver.DriverConnOpt) *Dcron{
	return NewDcron(serverName,"redis",dataSourceOption)

}
func NewDcron(serverName,driverName string, dataSourceOption driver.DriverConnOpt) *Dcron{

	dcron := new(Dcron)
	dcron.ServerName = serverName
	dcron.cr =	cron.New()
	dcron.jobs = make(map[string]*JobWarpper)
	dcron.nodePool = newNodePool(serverName,driverName,dataSourceOption)
	return dcron
}

func(this *Dcron)AddFunc(jobName,cronStr string,cmd func()){

	job := JobWarpper{
		Name:jobName,
		CronStr:cronStr,
		Func:cmd,
		Dcron:this,
	}

	this.cr.AddJob(cronStr,job)
}

func(this *Dcron)allowThisNodeRun(jobName string) bool{
	fmt.Println("nodeid",this.nodePool.NodeId)
	fmt.Println("allow node id",this.nodePool.PickNodeByJobName(jobName))
	return this.nodePool.NodeId == this.nodePool.PickNodeByJobName(jobName)
}



func(this *Dcron)Start(){
	this.cr.Start()
}

func(this *Dcron)Stop(){
	this.cr.Stop()
}




