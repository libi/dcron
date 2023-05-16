package special_jobs

import (
	"strings"
	"testing"
	"time"
)

func TestJobBashRun(t *testing.T) {
	call := func(bytes []byte, err error) {
		if err != nil {
			t.Logf("bash running error: %v \n", err)
		}
		t.Logf("bash output: %s \n", bytes)
	}

	bashJob1 := BashJob{
		Cmd:        "ping 127.0.0.1",
		Timeout:    time.Second * 5,
		OutputCall: call,
	}
	bashJob1.Run()

	bashJob2 := BashJob{
		Cmd:        "ps aux | grep go | wc -l",
		OutputCall: call,
	}
	bashJob2.Run()

	bashJob3 := BashJob{
		Cmd:        "date && sleep 10 && date",
		OutputCall: call,
	}
	bashJob3.Run()

	bashJob4 := BashJob{
		Cmd: "echo hello dcron",
		OutputCall: func(bytes []byte, err error) {
			if !strings.Contains(string(bytes), "hello dcron") {
				t.Errorf("echo hello dcron output [%s] not expected", bytes)
			}
		},
	}
	bashJob4.Run()
}
