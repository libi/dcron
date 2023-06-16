package dcron

import (
	"context"
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
	defaultDuration = 3 * time.Second

	dcronRunning = 1
	dcronStopped = 0

	dcronState_Steady  = "dcronState_Steady"
	dcronState_Upgrade = "dcronState_Upgrade"
)

type RecoverFuncType func(d *Dcron)

// Dcron is main struct
type Dcron struct {
	jobs      map[string]*JobWarpper
	jobsRWMut sync.Mutex

	ServerName string
	nodePool   INodePool
	running    int32

	logger  dlog.Logger
	logInfo bool

	nodeUpdateDuration time.Duration
	hashReplicas       int

	cr        *cron.Cron
	crOptions []cron.Option

	RecoverFunc RecoverFuncType

	recentJobs IRecentJobPacker
	state      atomic.Value
}

// NewDcron create a Dcron
func NewDcron(serverName string, driver driver.DriverV2, cronOpts ...cron.Option) *Dcron {
	dcron := newDcron(serverName)
	dcron.crOptions = cronOpts
	dcron.cr = cron.New(cronOpts...)
	dcron.running = dcronStopped
	dcron.nodePool = NewNodePool(serverName, driver, dcron.nodeUpdateDuration, dcron.hashReplicas, dcron.logger)
	return dcron
}

// NewDcronWithOption create a Dcron with Dcron Option
func NewDcronWithOption(serverName string, driver driver.DriverV2, dcronOpts ...Option) *Dcron {
	dcron := newDcron(serverName)
	for _, opt := range dcronOpts {
		opt(dcron)
	}

	dcron.cr = cron.New(dcron.crOptions...)
	dcron.nodePool = NewNodePool(serverName, driver, dcron.nodeUpdateDuration, dcron.hashReplicas, dcron.logger)
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

// AddJob  add a job
func (d *Dcron) AddJob(jobName, cronStr string, job Job) (err error) {
	return d.addJob(jobName, cronStr, nil, job)
}

// AddFunc add a cron func
func (d *Dcron) AddFunc(jobName, cronStr string, cmd func()) (err error) {
	return d.addJob(jobName, cronStr, cmd, nil)
}
func (d *Dcron) addJob(jobName, cronStr string, cmd func(), job Job) (err error) {
	d.logger.Infof("addJob '%s' : %s", jobName, cronStr)

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

func (d *Dcron) allowThisNodeRun(jobName string) (ok bool) {
	ok, err := d.nodePool.CheckJobAvailable(jobName)
	if err != nil {
		d.logger.Errorf("allow this node run error, err=%v", err)
		ok = false
		d.state.Store(NodePoolState_Upgrade)
	} else {
		d.state.Store(NodePoolState_Steady)
		if d.recentJobs != nil {
			go d.reRunRecentJobs(d.recentJobs.PopAllJobs())
		}
	}
	if d.recentJobs != nil {
		if d.state.Load().(string) == NodePoolState_Upgrade {
			d.recentJobs.AddJob(jobName, time.Now())
		}
	}
	return
}

// Start job
func (d *Dcron) Start() {
	// recover jobs before starting
	if d.RecoverFunc != nil {
		d.RecoverFunc(d)
	}
	if atomic.CompareAndSwapInt32(&d.running, dcronStopped, dcronRunning) {
		if err := d.startNodePool(); err != nil {
			atomic.StoreInt32(&d.running, dcronStopped)
			return
		}
		d.cr.Start()
		d.logger.Infof("dcron started, nodeID is %s", d.nodePool.GetNodeID())
	} else {
		d.logger.Infof("dcron have started")
	}
}

// Run Job
func (d *Dcron) Run() {
	// recover jobs before starting
	if d.RecoverFunc != nil {
		d.RecoverFunc(d)
	}
	if atomic.CompareAndSwapInt32(&d.running, dcronStopped, dcronRunning) {
		if err := d.startNodePool(); err != nil {
			atomic.StoreInt32(&d.running, dcronStopped)
			return
		}
		d.logger.Infof("dcron running, nodeID is %s", d.nodePool.GetNodeID())
		d.cr.Run()
	} else {
		d.logger.Infof("dcron already running")
	}
}

func (d *Dcron) startNodePool() error {
	if err := d.nodePool.Start(context.Background()); err != nil {
		d.logger.Errorf("dcron start node pool error %+v", err)
		return err
	}
	return nil
}

// Stop job
func (d *Dcron) Stop() {
	tick := time.NewTicker(time.Millisecond)
	d.nodePool.Stop(context.Background())
	for range tick.C {
		if atomic.CompareAndSwapInt32(&d.running, dcronRunning, dcronStopped) {
			d.cr.Stop()
			d.logger.Infof("dcron stopped")
			return
		}
	}
}

func (d *Dcron) reRunRecentJobs(jobNames []string) {
	d.logger.Infof("reRunRecentJobs: length=%d", len(jobNames))
	for _, jobName := range jobNames {
		if job, ok := d.jobs[jobName]; ok {
			if ok, _ := d.nodePool.CheckJobAvailable(jobName); ok {
				job.Execute()
			}
		}
	}
}
