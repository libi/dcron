package special_jobs

import (
	"time"
)

// NewBashJob New a bash shell job
func NewBashJob(cmd string, execTimeout time.Duration, outputCall func([]byte, error)) *BashJob {
	return &BashJob{Cmd: cmd, Timeout: execTimeout, OutputCall: outputCall}
}
