package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/libi/dcron"
	"github.com/libi/dcron/dlog"
	"github.com/libi/dcron/driver"
)

const (
	DriverType_REDIS = "redis"
	DriverType_ETCD  = "etcd"
)

var (
	addr       = flag.String("addr", "127.0.0.1:6379", "the addr of driver service")
	serverName = flag.String("server_name", "server", "the server name of dcron in this process")
	subId      = flag.String("sub_id", "1", "this process sub id in this server")
	jobNumber  = flag.Int("jobnumber", 3, "there number of cron job")
)

type WriteJob struct {
	Id     int
	Logger dlog.Logger
}

func (wj *WriteJob) Run() {
	filename := "tmpfile" + strconv.Itoa(wj.Id)
	// open file and append some msg to it.
	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_APPEND, 0755)
	if err != nil {
		wj.Logger.Errorf("err=%v", err)
		return
	}
	n, err := f.WriteString(
		fmt.Sprintf("sub_id=%s, time=%s, append string=%s\n", *subId, time.Now().String(), uuid.New().String()),
	)
	if err != nil {
		wj.Logger.Errorf("err=%v", err)
		return
	}
	wj.Logger.Infof("write length=%d", n)
	err = f.Close()
	if err != nil {
		wj.Logger.Errorf("err=%v", err)
		return
	}
}

func main() {
	flag.Parse()
	var err error

	redisCli := redis.NewClient(&redis.Options{
		Addr: *addr,
	})
	driver := driver.NewRedisDriver(redisCli)
	logger := &dlog.StdLogger{
		Log:        log.New(os.Stdout, "["+*subId+"]", log.LstdFlags),
		LogVerbose: true,
	}
	dcron := dcron.NewDcronWithOption(
		*serverName,
		driver,
		dcron.WithLogger(logger),
		dcron.WithHashReplicas(10),
		dcron.WithNodeUpdateDuration(time.Second*10),
	)
	for i := 1; i <= *jobNumber; i++ {
		job := &WriteJob{
			Id:     i,
			Logger: logger,
		}
		err = dcron.AddJob("write-task"+strconv.Itoa(i), "* * * * *", job)
		if err != nil {
			panic(err)
		}
	}
	dcron.Start()

	// run forever
	tick := time.NewTicker(time.Hour)
	for range tick.C {
	}
}
