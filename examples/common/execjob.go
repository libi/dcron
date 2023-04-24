package common

import (
	"bytes"
	"encoding/json"
	"log"
	"os/exec"
)

type ExecJob struct {
	Cron    string `json:"cron"`
	Command string `json:"command"`
}

func (execjob *ExecJob) GetCron() string {
	return execjob.Cron
}
func (execjob *ExecJob) Serialize() ([]byte, error) {
	return json.Marshal(execjob)
}
func (execjob *ExecJob) UnSerialize(b []byte) error {
	return json.Unmarshal(b, execjob)
}

func (execjob *ExecJob) Run() {
	cmd := exec.Command("bash", "-c", execjob.Command)
	var (
		stdout bytes.Buffer
	)
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		log.Println(err)
	}
	log.Println(stdout.String())
}
