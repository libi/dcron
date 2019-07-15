package dcron

import (
	"github.com/LibiChai/dcron/driver"
	"github.com/robfig/cron"
	"sync"
)

//Dcron is main struct
type Dcron struct {
	jobs       map[string]*JobWarpper
	mu         sync.RWMutex
	cr         *cron.Cron
	ServerName string
	nodePool   *NodePool
}

//NewDcronUseRedis use redis driver
func NewDcronUseRedis(serverName string, dataSourceOption driver.DriverConnOpt) *Dcron {
	return NewDcron(serverName, "redis", dataSourceOption)

}

//NewDcron create a Dcron
func NewDcron(serverName, driverName string, dataSourceOption driver.DriverConnOpt) *Dcron {

	dcron := new(Dcron)
	dcron.ServerName = serverName
	dcron.cr = cron.New()
	dcron.jobs = make(map[string]*JobWarpper)
	dcron.nodePool = newNodePool(serverName, driverName, dataSourceOption)
	return dcron
}

//AddFunc add a job
func (d *Dcron) AddFunc(jobName, cronStr string, cmd func()) (err error) {

	job := JobWarpper{
		Name:    jobName,
		CronStr: cronStr,
		Func:    cmd,
		Dcron:   d,
	}

	return d.cr.AddJob(cronStr, job)
}

func (d *Dcron) allowThisNodeRun(jobName string) bool {
	return d.nodePool.NodeID == d.nodePool.PickNodeByJobName(jobName)
}

//Start start job
func (d *Dcron) Start() {
	d.cr.Start()
}

//Stop stop job
func (d *Dcron) Stop() {
	d.cr.Stop()
}
