package main

import (
	"context"
	"flag"

	examplesCommon "github.com/libi/dcron/examples/common"
	"github.com/redis/go-redis/v9"
)

var (
	addr = flag.String("addr", "127.0.0.1:6379", "redis addr")
	pwd  = flag.String("pwd", "123456", "redis password")
	key  = flag.String("key", "key", "stable job key")
)

func Store(cli *redis.Client, jobName string, job *examplesCommon.ExecJob) {
	b, _ := job.Serialize()
	_ = cli.HSet(context.Background(), *key, jobName, b)
}

func main() {
	flag.Parse()
	opts := &redis.Options{
		Addr:     *addr,
		Password: *pwd,
	}
	cli := redis.NewClient(opts)
	Store(cli, "job1", &examplesCommon.ExecJob{
		Cron:    "@every 10s",
		Command: "date",
	})

	Store(cli, "job2", &examplesCommon.ExecJob{
		Cron:    "15 * * * * *",
		Command: "df -h",
	})
}
