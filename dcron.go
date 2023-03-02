package dcron

import (
	"errors"
	"log"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/libi/dcron/dlog"
	"github.com/libi/dcron/driver"
	"github.com/robfig/cron/v3"
)

const (
	defaultReplicas = 50
	defaultDuration = time.Second
)

const (
	dcronRunning = 1
	dcronStopped = 0
)

// Dcron is main struct
type Dcron struct {
	jobs      map[string]*JobWarpper
	jobsRWMut sync.Mutex

	ServerName string
	nodePool   *NodePool
	running    int32

	logger  dlog.Logger
	logInfo bool

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
	dcron.running = dcronStopped
	var err error
	dcron.nodePool, err = newNodePool(serverName, driver, dcron, dcron.nodeUpdateDuration, dcron.hashReplicas)
	if err != nil {
		dcron.logger.Errorf("ERR: %s", err.Error())
		return nil
	}
	return dcron
}

// NewDcronWithOption create a Dcron with Dcron Option
func NewDcronWithOption(serverName string, driver driver.Driver, dcronOpts ...Option) *Dcron {
	dcron := newDcron(serverName)
	for _, opt := range dcronOpts {
		opt(dcron)
	}

	dcron.cr = cron.New(dcron.crOptions...)
	var err error
	dcron.nodePool, err = newNodePool(serverName, driver, dcron, dcron.nodeUpdateDuration, dcron.hashReplicas)
	if err != nil {
		dcron.logger.Errorf("ERR: %s", err.Error())
		return nil
	}

	return dcron
}

func newDcron(serverName string) *Dcron {
	return &Dcron{
		ServerName: serverName,
		logger: &dlog.StdLogger{
			Log: log.New(os.Stdout, "[dcron] ", log.LstdFlags),
		},
		jobs:               make(map[string]*JobWarpper),
		crOptions:          make([]cron.Option, 0),
		nodeUpdateDuration: defaultDuration,
		hashReplicas:       defaultReplicas,
	}
}

// SetLogger set dcron logger
func (d *Dcron) SetLogger(logger dlog.Logger) {
	d.logger = logger
}

// GetLogger get dcron logger
func (d *Dcron) GetLogger() dlog.Logger {
	return d.logger
}

func (d *Dcron) AddStableJob(jobName string, job StableJob) (err error) {
	if !d.nodePool.Driver.SupportStableJob() {
		err = errors.New("stable job mode not supported")
		return
	}
	b, err := job.Serialize()
	if err != nil {
		d.logger.Errorf("serialize job failed, %v", err)
		return
	}
	err = d.nodePool.Driver.Store(d.ServerName, jobName, b)
	if err != nil {
		d.logger.Errorf("store job failed, %v", err)
		return
	}
	err = d.addJob(jobName, job.GetCron(), nil, job)
	if err != nil {
		d.logger.Errorf("add stableJob failed, %v", err)
		return
	}
	return
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
	d.logger.Infof("addJob '%s' :  %s", jobName, cronStr)

	d.jobsRWMut.Lock()
	defer d.jobsRWMut.Unlock()
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
	d.jobsRWMut.Lock()
	defer d.jobsRWMut.Unlock()

	if job, ok := d.jobs[jobName]; ok {
		delete(d.jobs, jobName)
		d.cr.Remove(job.ID)
	}
}

func (d *Dcron) allowThisNodeRun(jobName string) bool {
	allowRunNode := d.nodePool.PickNodeByJobName(jobName)
	d.logger.Infof("job '%s' running in node %s", jobName, allowRunNode)
	if allowRunNode == "" {
		d.logger.Errorf("node pool is empty")
		return false
	}
	return d.nodePool.NodeID == allowRunNode
}

// Start job
func (d *Dcron) Start() {
	if atomic.CompareAndSwapInt32(&d.running, dcronStopped, dcronRunning) {
		if err := d.startNodePool(); err != nil {
			atomic.StoreInt32(&d.running, dcronStopped)
			return
		}
		d.cr.Start()
		d.logger.Infof("dcron started , nodeID is %s", d.nodePool.NodeID)
	} else {
		d.logger.Infof("dcron have started")
	}
}

// Run Job
func (d *Dcron) Run() {
	if atomic.CompareAndSwapInt32(&d.running, dcronStopped, dcronRunning) {
		if err := d.startNodePool(); err != nil {
			atomic.StoreInt32(&d.running, dcronStopped)
			return
		}

		d.logger.Infof("dcron running nodeID is %s", d.nodePool.NodeID)
		d.cr.Run()
	} else {
		d.logger.Infof("dcron already running")
	}
}

func (d *Dcron) startNodePool() error {
	if err := d.nodePool.StartPool(); err != nil {
		d.logger.Errorf("dcron start node pool error %+v", err)
		return err
	}
	return nil
}

// Stop job
func (d *Dcron) Stop() {
	tick := time.NewTicker(time.Millisecond)
	for range tick.C {
		if atomic.CompareAndSwapInt32(&d.running, dcronRunning, dcronStopped) {
			d.cr.Stop()
			d.logger.Infof("dcron stopped")
			return
		}
	}
}
