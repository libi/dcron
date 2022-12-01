package dcron

import (
	"errors"
	"github.com/libi/dcron/driver"
	"github.com/robfig/cron/v3"
	"log"
	"os"
	"sync/atomic"
	"time"
)

const (
	defaultReplicas = 50
	defaultDuration = time.Second
)

const (
	dcronRunning = 1
	dcronStoped  = 0
)

// Dcron is main struct
type Dcron struct {
	jobs       map[string]*JobWarpper
	ServerName string
	nodePool   *NodePool
	running    int32

	logger interface{ Printf(string, ...interface{}) }

	nodeUpdateDuration time.Duration
	hashReplicas       int

	cr        *cron.Cron
	crOptions []cron.Option
}

// NewDcron create a Dcron
func NewDcron(serverName string, driver driver.Driver, cronOpts ...cron.Option) *Dcron {
	dcron := newDcron(serverName)
	dcron.crOptions = cronOpts
	dcron.cr = cron.New(cronOpts...)
	dcron.running = dcronStoped
	dcron.nodePool = newNodePool(serverName, driver, dcron, dcron.nodeUpdateDuration, dcron.hashReplicas)
	return dcron
}

// NewDcronWithOption create a Dcron with Dcron Option
func NewDcronWithOption(serverName string, driver driver.Driver, dcronOpts ...Option) *Dcron {
	dcron := newDcron(serverName)
	for _, opt := range dcronOpts {
		opt(dcron)
	}

	dcron.cr = cron.New(dcron.crOptions...)
	dcron.nodePool = newNodePool(serverName, driver, dcron, dcron.nodeUpdateDuration, dcron.hashReplicas)
	return dcron
}

func newDcron(serverName string) *Dcron {
	return &Dcron{
		ServerName:         serverName,
		logger:             log.New(os.Stdout, "[dcron] ", log.LstdFlags),
		jobs:               make(map[string]*JobWarpper),
		crOptions:          make([]cron.Option, 0),
		nodeUpdateDuration: defaultDuration,
		hashReplicas:       defaultReplicas,
	}
}

// SetLogger set dcron logger
func (d *Dcron) SetLogger(logger *log.Logger) {
	d.logger = logger
}

// GetLogger get dcron logger
func (d *Dcron) GetLogger() interface{ Printf(string, ...interface{}) } {
	return d.logger
}

func (d *Dcron) info(format string, v ...interface{}) {
	d.logger.Printf("INFO: "+format, v...)
}
func (d *Dcron) err(format string, v ...interface{}) {
	d.logger.Printf("ERR: "+format, v...)
}

// AddJob  add a job
func (d *Dcron) AddJob(jobName, cronStr string, job Job) (err error) {
	return d.addJob(jobName, cronStr, nil, job)
}

// AddFunc add a cron func
func (d *Dcron) AddFunc(jobName, cronStr string, cmd func()) (err error) {
	return d.addJob(jobName, cronStr, cmd, nil)
}
func (d *Dcron) addJob(jobName, cronStr string, cmd func(), job Job) (err error) {
	d.info("addJob '%s' :  %s", jobName, cronStr)
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

	return nil
}

// Remove Job
func (d *Dcron) Remove(jobName string) {
	if job, ok := d.jobs[jobName]; ok {
		delete(d.jobs, jobName)
		d.cr.Remove(job.ID)
	}
}

func (d *Dcron) allowThisNodeRun(jobName string) bool {
	allowRunNode := d.nodePool.PickNodeByJobName(jobName)
	d.info("job '%s' running in node %s", jobName, allowRunNode)
	if allowRunNode == "" {
		d.err("node pool is empty")
		return false
	}
	return d.nodePool.NodeID == allowRunNode
}

// Start job
func (d *Dcron) Start() {
	if atomic.CompareAndSwapInt32(&d.running, dcronStoped, dcronRunning) {
		if err := d.startNodePool(); err != nil {
			atomic.StoreInt32(&d.running, dcronStoped)
			return
		}
		d.cr.Start()
		d.info("dcron started , nodeID is %s", d.nodePool.NodeID)
	} else {
		d.info("dcron have started")
	}
}

// Run Job
func (d *Dcron) Run() {
	if atomic.CompareAndSwapInt32(&d.running, dcronStoped, dcronRunning) {
		if err := d.startNodePool(); err != nil {
			atomic.StoreInt32(&d.running, dcronStoped)
			return
		}

		d.info("dcron running nodeID is %s", d.nodePool.NodeID)
		d.cr.Run()
	} else {
		d.info("dcron already running")
	}
}

func (d *Dcron) startNodePool() error {
	if err := d.nodePool.StartPool(); err != nil {
		d.err("dcron start node pool error %+v", err)
		return err
	}
	return nil
}

// Stop job
func (d *Dcron) Stop() {
	for {
		if atomic.CompareAndSwapInt32(&d.running, dcronRunning, dcronStoped) {
			d.cr.Stop()
			d.info("dcron stopped")
			return
		}
		time.Sleep(time.Millisecond)
	}
}
