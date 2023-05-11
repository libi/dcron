package special_jobs

import (
	"bytes"
	"context"
	"github.com/libi/dcron/dlog"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

type BashJob struct {
	Cmds    []string
	Timeout time.Duration
	Logger  dlog.Logger
}

func (job BashJob) splitCmd(i int) []string {
	return strings.SplitN(job.Cmds[i], " ", 2)
}

func (job BashJob) Run() {
	var cmd *exec.Cmd
	var stdOut bytes.Buffer
	var stdErr bytes.Buffer
	var cmdStr string
	sysType := runtime.GOOS

	for i := range job.Cmds {
		cmds := job.splitCmd(i)
		c := strings.Join(cmds, " ")
		switch sysType {
		case "linux", "darwin":
			if job.Timeout > 0 {
				ctx, cancel := context.WithTimeout(context.Background(), job.Timeout)
				defer cancel()
				cmd = exec.CommandContext(ctx, "sh", "-c", c)
			} else {
				cmd = exec.Command("sh", "-c", c)
			}
		case "windows":
			if job.Timeout > 0 {
				ctx, cancel := context.WithTimeout(context.Background(), job.Timeout)
				defer cancel()
				cmd = exec.CommandContext(ctx, "sh", "-c", c)
			} else {
				cmd = exec.Command("PowerShell", "-Command", c)
			}
		default:
			job.Logger.Errorf("cmd os not supported: %s", sysType)
			return
		}
		cmd.Stdout = &stdOut
		cmd.Stderr = &stdErr

		if i > 0 {
			cmdStr = cmdStr + "\n" + c
		} else {
			cmdStr = c
		}
		if err := cmd.Start(); err != nil {
			// 错误处理
			job.Logger.Errorf("cmd run failed: %v", err)
			break
		}

		if err := cmd.Wait(); err != nil {
			// 错误处理
			if p, _ := os.FindProcess(cmd.Process.Pid); p != nil {
				// 无法杀死任务启动的子进程
				p.Kill()
			}
			job.Logger.Errorf("cmd run failed: %v", err)
			// 告警之类
			break
		}
	}

	job.Logger.Infof("------ " + cmdStr + " ----\n" + stdOut.String() + " ----\n" + stdErr.String())
	return
}
