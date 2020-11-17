package dcron

import (
	"github.com/robfig/cron/v3"
	"log"
	"time"
)

type Option func(*Dcron)

// WithLogger both set dcron and cron logger.
func WithLogger(logger *log.Logger) Option {
	return func(dcron *Dcron) {
		//set dcron logger
		dcron.logger = logger
		//set cron logger
		f := cron.WithLogger(cron.PrintfLogger(logger))
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

//CronOptionLocation is warp cron with location
func CronOptionLocation(loc *time.Location) Option {
	return func(dcron *Dcron) {
		f := cron.WithLocation(loc)
		dcron.crOptions = append(dcron.crOptions, f)
	}
}

//CronOptionSeconds is warp cron with seconds
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
