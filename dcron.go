package dcron

import (
	"errors"
	. "github.com/LibiChai/dcron/driver"
	"github.com/robfig/cron/v3"
	"sync"
)

//Dcron is main struct
type Dcron struct {
	jobs       map[string]*JobWarpper
	mu         sync.RWMutex
	cr         *cron.Cron
	ServerName string
	nodePool   *NodePool
	isRun      bool
}

//NewDcron create a Dcron
func NewDcron(serverName string, driver Driver, opts ...cron.Option) *Dcron {

	dcron := new(Dcron)
	dcron.ServerName = serverName
	dcron.cr = cron.New(opts...)
	dcron.jobs = make(map[string]*JobWarpper)
	dcron.nodePool = newNodePool(serverName, driver, dcron)
	return dcron
}

//AddJob  add a job
func (d *Dcron) AddJob(jobName, cronStr string, job Job) (err error) {
	return d.addJob(jobName, cronStr, nil, job)
}

//AddFunc add a cron func
func (d *Dcron) AddFunc(jobName, cronStr string, cmd func()) (err error) {
	return d.addJob(jobName, cronStr, cmd, nil)
}
func (d *Dcron) addJob(jobName, cronStr string, cmd func(), job Job) (err error) {

	if _, ok := d.jobs[jobName]; ok {
		return errors.New("jobName already exist")
	}
	innerJob := JobWarpper{
		Name:    jobName,
		CronStr: cronStr,
		Func:    cmd,
		Job:     job,
		Dcron:   d,
	}
	entryID, err := d.cr.AddJob(cronStr, innerJob)
	if err != nil {
		return err
	}
	innerJob.ID = entryID
	d.jobs[jobName] = &innerJob
	return err

}

// Remove Job
func (d *Dcron) Remove(jobName string) {
	if job, ok := d.jobs[jobName]; ok {
		delete(d.jobs, jobName)
		d.cr.Remove(job.ID)
	}
}

func (d *Dcron) allowThisNodeRun(jobName string) bool {
	return d.nodePool.NodeID == d.nodePool.PickNodeByJobName(jobName)
}

//Start start job
func (d *Dcron) Start() {
	d.isRun = true
	d.cr.Start()
}

// Run Job
func (d *Dcron) Run() {
	d.isRun = true
	d.cr.Run()
}

//Stop stop job
func (d *Dcron) Stop() {
	d.isRun = false
	d.cr.Stop()
}
