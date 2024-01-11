package dcron

import (
	"context"
	"errors"
	"log"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/libi/dcron/cron"
	"github.com/libi/dcron/dlog"
	"github.com/libi/dcron/driver"
)

const (
	defaultReplicas = 50
	defaultDuration = 3 * time.Second

	dcronRunning = 1
	dcronStopped = 0

	dcronStateSteady  = "dcronStateSteady"
	dcronStateUpgrade = "dcronStateUpgrade"
)

var (
	ErrJobExist     = errors.New("jobName already exist")
	ErrJobNotExist  = errors.New("jobName not exist")
	ErrJobWrongNode = errors.New("job is not running in this node")
)

type RecoverFuncType func(d *Dcron)

// Dcron is main struct
type Dcron struct {
	jobs      map[string]*JobWarpper
	jobsRWMut sync.RWMutex

	ServerName string
	nodePool   INodePool
	running    int32

	logger dlog.Logger

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
	return d.addJob(jobName, cronStr, job)
}

// AddFunc add a cron func
func (d *Dcron) AddFunc(jobName, cronStr string, cmd func()) (err error) {
	return d.addJob(jobName, cronStr, cron.FuncJob(cmd))
}
func (d *Dcron) addJob(jobName, cronStr string, job Job) (err error) {
	d.logger.Infof("addJob '%s' : %s", jobName, cronStr)

	d.jobsRWMut.Lock()
	defer d.jobsRWMut.Unlock()
	if _, ok := d.jobs[jobName]; ok {
		return ErrJobExist
	}
	innerJob := JobWarpper{
		Name:    jobName,
		CronStr: cronStr,
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

// Remove Job by jobName
func (d *Dcron) Remove(jobName string) {
	d.jobsRWMut.Lock()
	defer d.jobsRWMut.Unlock()

	if job, ok := d.jobs[jobName]; ok {
		delete(d.jobs, jobName)
		d.cr.Remove(job.ID)
	}
}

// Get job by jobName
// if this jobName not exist, will return error.
//
//	if `thisNodeOnly` is true
//		if this job is not available in this node, will return error.
//	otherwise return the struct of JobWarpper whose name is jobName.
func (d *Dcron) GetJob(jobName string, thisNodeOnly bool) (*JobWarpper, error) {
	d.jobsRWMut.RLock()
	defer d.jobsRWMut.RUnlock()

	job, ok := d.jobs[jobName]
	if !ok {
		d.logger.Warnf("job: %s, not exist", jobName)
		return nil, ErrJobNotExist
	}
	if !thisNodeOnly {
		return job, nil
	}
	isRunningHere, err := d.nodePool.CheckJobAvailable(jobName)
	if err != nil {
		return nil, err
	}
	if isRunningHere {
		return nil, ErrJobWrongNode
	}
	return job, nil
}

// Get job list.
//
//	if `thisNodeOnly` is true
//		return all jobs available in this node.
//	otherwise return all jobs added to dcron.
//
// we never return nil. If there is no job.
// this func will return an empty slice.
func (d *Dcron) GetJobs(thisNodeOnly bool) []*JobWarpper {
	d.jobsRWMut.RLock()
	defer d.jobsRWMut.RUnlock()

	ret := make([]*JobWarpper, 0)
	for _, v := range d.jobs {
		var (
			isRunningHere bool
			ok            bool = true
			err           error
		)
		if thisNodeOnly {
			isRunningHere, err = d.nodePool.CheckJobAvailable(v.Name)
			if err != nil {
				continue
			}
			ok = isRunningHere
		}
		if ok {
			ret = append(ret, v)
		}
	}
	return ret
}

func (d *Dcron) allowThisNodeRun(jobName string) (ok bool) {
	ok, err := d.nodePool.CheckJobAvailable(jobName)
	if err != nil {
		d.logger.Errorf("allow this node run error, err=%v", err)
		ok = false
		d.state.Store(dcronStateUpgrade)
	} else {
		d.state.Store(dcronStateSteady)
		if d.recentJobs != nil {
			go d.reRunRecentJobs(d.recentJobs.PopAllJobs())
		}
	}
	if d.recentJobs != nil {
		if d.state.Load().(string) == dcronStateUpgrade {
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

func (d *Dcron) NodeID() string {
	return d.nodePool.GetNodeID()
}
