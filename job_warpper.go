package dcron

import "github.com/robfig/cron/v3"

// Job Interface
type Job interface {
	Run()
}

// This type of Job will be
// recovered in a node of service
// restarting.
type StableJob interface {
	Job
	GetCron() string
	Serialize() ([]byte, error)
	UnSerialize([]byte) error
}

//JobWarpper is a job warpper
type JobWarpper struct {
	ID      cron.EntryID
	Dcron   *Dcron
	Name    string
	CronStr string
	Func    func()
	Job     Job
}

//Run is run job
func (job JobWarpper) Run() {
	//如果该任务分配给了这个节点 则允许执行
	if job.Dcron.allowThisNodeRun(job.Name) {
		if job.Func != nil {
			job.Func()
		}
		if job.Job != nil {
			job.Job.Run()
		}
	}
}
