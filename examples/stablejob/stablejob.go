package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/libi/dcron"
	"github.com/libi/dcron/dlog"
	v2 "github.com/libi/dcron/driver/v2"
	examplesCommon "github.com/libi/dcron/examples/common"
)

type EnvConfig struct {
	RedisAddr     string
	RedisPassword string
	ServerName    string
	JobStoreKey   string
}

func defaultString(targetStr string, defaultStr string) string {
	if targetStr == "" {
		return defaultStr
	}
	return targetStr
}

func (ec *EnvConfig) LoadEnv() {
	ec.RedisAddr = defaultString(os.Getenv("redis_addr"), "127.0.0.1:6379")
	ec.RedisPassword = defaultString(os.Getenv("redis_password"), "123456")
	ec.ServerName = defaultString(os.Getenv("server_name"), "dcronsvr")
	ec.JobStoreKey = defaultString(os.Getenv("job_store_key"), "job_store_key")
}

var IEnv EnvConfig

func main() {
	logger := log.New(os.Stdout, "", log.LstdFlags)
	IEnv.LoadEnv()

	redisOpts := &redis.Options{
		Addr:     IEnv.RedisAddr,
		Password: IEnv.RedisPassword,
	}

	redisCli := redis.NewClient(redisOpts)
	drv := v2.NewRedisDriver(redisCli)
	dcronInstance := dcron.NewDcronWithOption(IEnv.ServerName, drv,
		dcron.WithLogger(&dlog.StdLogger{
			Log: logger,
		}),
		dcron.CronOptionSeconds(),
		dcron.WithHashReplicas(10),
		dcron.WithRecoverFunc(func(d *dcron.Dcron) {
			cli := redis.NewClient(redisOpts)
			defer cli.Close()
			results, err := cli.HGetAll(context.Background(), IEnv.JobStoreKey).Result()
			if err != nil {
				logger.Println(err)
				return
			}
			logger.Printf("sizeof stablejobs containers=%d, job_store_key=%s", len(results), IEnv.JobStoreKey)
			for k, v := range results {
				logger.Printf("jobName=%s, jobBody=%s", k, v)
				job := &examplesCommon.ExecJob{}
				err = job.UnSerialize([]byte(v))
				if err != nil {
					logger.Printf("unserialize job error: %v", err)
					continue
				}
				err = d.AddJob(k, job.GetCron(), job)
				if err != nil {
					logger.Printf("add job error: %v", err)
					continue
				}
			}
		}),
	)
	dcronInstance.Start()
	defer dcronInstance.Stop()
	// run forever
	tick := time.NewTicker(time.Hour)
	for range tick.C {
	}
}
