package dcron

import "time"

// IRecentJobPacker
// this is an interface which be used to
// pack the jobs running in the cluster state
// is `unstable`.
// like some nodes broken or new nodes were add.
type IRecentJobPacker interface {
	// goroutine safety.
	// Add a job to packer
	// will save recent jobs (like 2 * heartbeat duration)
	AddJob(jobName string, t time.Time) error

	// goroutine safety.
	// Pop out all jobs in packer
	PopAllJobs() (jobNames []string)
}
