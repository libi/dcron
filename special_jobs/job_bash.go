package special_jobs

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"runtime"
	"time"
)

type BashJob struct {
	Cmd        string
	Timeout    time.Duration
	OutputCall func([]byte, error)
}

func (job BashJob) Run() {
	var cmd *exec.Cmd
	sysType := runtime.GOOS
	switch sysType {
	case "linux", "darwin":
		if job.Timeout > 0 {
			ctx, cancel := context.WithTimeout(context.Background(), job.Timeout)
			defer cancel()
			cmd = exec.CommandContext(ctx, "sh", "-c", job.Cmd)
		} else {
			cmd = exec.Command("sh", "-c", job.Cmd)
		}
	case "windows":
		if job.Timeout > 0 {
			ctx, cancel := context.WithTimeout(context.Background(), job.Timeout)
			defer cancel()
			cmd = exec.CommandContext(ctx, "PowerShell", "-Command", job.Cmd)
		} else {
			cmd = exec.Command("PowerShell", "-Command", job.Cmd)
		}
	default:
		job.OutputCall(nil, errors.New("unsupported bash os"))
		return
	}

	output, err := cmd.Output()

	if err == context.DeadlineExceeded {
		// timeout killing
		if p, _ := os.FindProcess(cmd.Process.Pid); p != nil {
			p.Kill()
		}
	}

	job.OutputCall(output, err)
}
