package v2

import (
	"context"
	"fmt"
	"log"
	"sort"
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
	nodesChan   chan []string
	prevNodes   []string
}

func NewRedisDriver(redisClient *redis.Client) driver.DriverV2 {
	return &RedisDriver{
		c: redisClient,
		logger: &dlog.StdLogger{
			Log: log.Default(),
		},
		nodesChan: make(chan []string, DefaultNodesChanLength),
		prevNodes: make([]string, 0),
	}
}

func (rd *RedisDriver) Init(serviceName string, timeout time.Duration, logger dlog.Logger) {
	rd.serviceName = serviceName
	rd.timeout = timeout
	if logger != nil {
		rd.logger = logger
	}
	rd.nodeID = GetNodeId(rd.serviceName)
}

func (rd *RedisDriver) Start() (nodesChan chan []string, err error) {
	// register
	err = rd.registerServiceNode()
	// heartbeat timer
	go rd.heartBeat()
	// update service nodes
	rd.updateNodes()
	// go update timer.
	go rd.updateNodesTimer()
	return rd.nodesChan, err
}

// private function

func (rd *RedisDriver) updateNodesTimer() {
	tick := time.NewTicker(rd.timeout / 2)
	for range tick.C {
		rd.updateNodes()
	}
}

func (rd *RedisDriver) updateNodes() {
	nowNodes, err := rd.getServiceNodeList()
	if err != nil {
		rd.logger.Errorf("get service node list err, err=%v", err)
		return
	}
	sort.Strings(nowNodes)
	if EqualStringSlice(rd.prevNodes, nowNodes) {
		return
	}
	rd.prevNodes = nowNodes
	rd.nodesChan <- rd.prevNodes
}

func (rd *RedisDriver) heartBeat() {
	tick := time.NewTicker(rd.timeout / 2)
	for range tick.C {
		keyExist, err := rd.c.Expire(context.Background(), rd.nodeID, rd.timeout).Result()
		if err != nil {
			rd.logger.Errorf("redis expire error %+v", err)
			continue
		}
		if !keyExist {
			if err := rd.registerServiceNode(); err != nil {
				rd.logger.Errorf("register service node error %+v", err)
			}
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

func (rd *RedisDriver) getServiceNodeList() (nodesList []string, err error) {
	mathStr := fmt.Sprintf("%s*", GetKeyPre(rd.serviceName))
	return rd.scan(mathStr)
}
