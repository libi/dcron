package v2

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/libi/dcron/dlog"
	"github.com/libi/dcron/driver"
)

type RedisDriver struct {
	c           *redis.Client
	serviceName string
	nodeID      string
	timeout     time.Duration
	logger      dlog.Logger
	started     bool
	stopChan    chan interface{}

	sync.Mutex
}

func NewRedisDriver(redisClient *redis.Client) driver.DriverV2 {
	rd := &RedisDriver{
		c: redisClient,
		logger: &dlog.StdLogger{
			Log: log.Default(),
		},
	}
	rd.started = false
	return rd
}

func (rd *RedisDriver) Init(serviceName string, timeout time.Duration, logger dlog.Logger) {
	rd.serviceName = serviceName
	rd.timeout = timeout
	if logger != nil {
		rd.logger = logger
	}
	rd.nodeID = GetNodeId(rd.serviceName)
}

func (rd *RedisDriver) NodeID() string {
	return rd.nodeID
}

func (rd *RedisDriver) Start() (err error) {
	rd.Lock()
	defer rd.Unlock()
	if rd.started {
		err = errors.New("this driver is started")
		return
	}
	rd.stopChan = make(chan interface{}, 1)
	rd.started = true
	// register
	err = rd.registerServiceNode()
	if err != nil {
		return
	}
	// heartbeat timer
	go rd.heartBeat()
	return
}

func (rd *RedisDriver) Stop() (err error) {
	rd.Lock()
	defer rd.Unlock()
	close(rd.stopChan)
	return
}

func (rd *RedisDriver) GetNodes() (nodes []string, err error) {
	mathStr := fmt.Sprintf("%s*", GetKeyPre(rd.serviceName))
	return rd.scan(mathStr)
}

// private function

func (rd *RedisDriver) heartBeat() {
	tick := time.NewTicker(rd.timeout / 2)
	for {
		select {
		case <-tick.C:
			if err := rd.registerServiceNode(); err != nil {
				rd.logger.Errorf("register service node error %+v", err)
			}
		case <-rd.stopChan:
			if err := rd.c.Del(context.Background(), rd.nodeID, rd.nodeID).Err(); err != nil {
				rd.logger.Errorf("unregister service node error %+v", err)
			}
			return
		}
	}
}

func (rd *RedisDriver) registerServiceNode() error {
	return rd.c.SetEX(context.Background(), rd.nodeID, rd.nodeID, rd.timeout).Err()
}

func (rd *RedisDriver) scan(matchStr string) ([]string, error) {
	ret := make([]string, 0)
	ctx := context.Background()
	iter := rd.c.Scan(ctx, 0, matchStr, -1).Iterator()
	for iter.Next(ctx) {
		err := iter.Err()
		if err != nil {
			return nil, err
		}
		ret = append(ret, iter.Val())
	}
	return ret, nil
}
