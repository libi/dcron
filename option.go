package dcron

import (
	"time"

	"github.com/libi/dcron/cron"
	"github.com/libi/dcron/dlog"
)

// Option is Dcron Option
type Option func(*Dcron)

// WithLogger both set dcron and cron logger.
func WithLogger(logger dlog.Logger) Option {
	return func(dcron *Dcron) {
		//set dcron logger
		dcron.logger = logger
		f := cron.WithLogger(logger)
		dcron.crOptions = append(dcron.crOptions, f)
	}
}

// WithNodeUpdateDuration set node update duration
func WithNodeUpdateDuration(d time.Duration) Option {
	return func(dcron *Dcron) {
		dcron.nodeUpdateDuration = d
	}
}

// WithHashReplicas set hashReplicas
func WithHashReplicas(d int) Option {
	return func(dcron *Dcron) {
		dcron.hashReplicas = d
	}
}

// CronOptionLocation is warp cron with location
func CronOptionLocation(loc *time.Location) Option {
	return func(dcron *Dcron) {
		f := cron.WithLocation(loc)
		dcron.crOptions = append(dcron.crOptions, f)
	}
}

// CronOptionSeconds is warp cron with seconds
func CronOptionSeconds() Option {
	return func(dcron *Dcron) {
		f := cron.WithSeconds()
		dcron.crOptions = append(dcron.crOptions, f)
	}
}

// CronOptionParser is warp cron with schedules.
func CronOptionParser(p cron.ScheduleParser) Option {
	return func(dcron *Dcron) {
		f := cron.WithParser(p)
		dcron.crOptions = append(dcron.crOptions, f)
	}
}

// CronOptionChain is Warp cron with chain
func CronOptionChain(wrappers ...cron.JobWrapper) Option {
	return func(dcron *Dcron) {
		f := cron.WithChain(wrappers...)
		dcron.crOptions = append(dcron.crOptions, f)
	}
}

// You can defined yourself recover function to make the
// job will be added to your dcron when the process restart
func WithRecoverFunc(recoverFunc RecoverFuncType) Option {
	return func(dcron *Dcron) {
		dcron.RecoverFunc = recoverFunc
	}
}

// You can use this option to start the recent jobs rerun
// after the cluster upgrading.
func WithClusterStable(timeWindow time.Duration) Option {
	return func(d *Dcron) {
		d.recentJobs = NewRecentJobPacker(timeWindow)
	}
}

func RunningLocally() Option {
	return func(d *Dcron) {
		d.runningLocally = true
	}
}
