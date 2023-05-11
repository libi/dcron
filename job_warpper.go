package dcron

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
)

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

// JobWarpper is a job warpper
type JobWarpper struct {
	ID      cron.EntryID
	Dcron   *Dcron
	Name    string
	CronStr string
	Func    func()
	Job     Job
	Timeout time.Duration

	Commands []string // 要执行的shell命令
	// 用于存储分隔后的任务
	cmd []string
}

// Run is run job
func (job JobWarpper) Run() {
	//如果该任务分配给了这个节点 则允许执行
	if job.Dcron.allowThisNodeRun(job.Name) {
		if job.Func != nil {
			job.Func()
		}
		if job.Job != nil {
			job.Job.Run()
		}
		if job.Commands != nil {
			job.CmdRun()
		}
	}
}

func (job *JobWarpper) splitCmd(i int) {
	job.cmd = strings.SplitN(job.Commands[i], " ", 2)
}

func (job *JobWarpper) CmdRun() (result string, err error) {
	var cmd *exec.Cmd
	var stdOut bytes.Buffer
	var stdErr bytes.Buffer
	var cmdStr string
	sysType := runtime.GOOS

	for i := range job.Commands {
		job.splitCmd(i)
		c := strings.Join(job.cmd, " ")
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
			err = errors.New("cmd os not supported")
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
			job.Dcron.logger.Errorf("cmd run failed: %v", err)
			break
		}

		if err := cmd.Wait(); err != nil {
			// 错误处理
			if p, _ := os.FindProcess(cmd.Process.Pid); p != nil {
				// 无法杀死任务启动的子进程
				p.Kill()
			}
			job.Dcron.logger.Errorf("cmd run failed: %v", err)
			// 告警之类
			break
		}
	}

	result = "------ " + cmdStr + " ----\n" + stdOut.String() + " ----\n" + stdErr.String()
	return
}
