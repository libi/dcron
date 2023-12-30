package driver

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/libi/dcron/dlog"
)

type RedisZSetDriver struct {
	c           *redis.Client
	serviceName string
	nodeID      string
	timeout     time.Duration
	logger      dlog.Logger
	started     bool

	// this context is used to define
	// the lifetime of this driver.
	runtimeCtx    context.Context
	runtimeCancel context.CancelFunc

	sync.Mutex
}

func newRedisZSetDriver(redisClient *redis.Client) *RedisZSetDriver {
	rd := &RedisZSetDriver{
		c: redisClient,
		logger: &dlog.StdLogger{
			Log: log.Default(),
		},
		timeout: redisDefaultTimeout,
	}
	rd.started = false
	return rd
}

func (rd *RedisZSetDriver) Init(serviceName string, opts ...Option) {
	rd.serviceName = serviceName
	rd.nodeID = GetNodeId(serviceName)
	for _, opt := range opts {
		rd.withOption(opt)
	}
}

func (rd *RedisZSetDriver) NodeID() string {
	return rd.nodeID
}

func (rd *RedisZSetDriver) GetNodes(ctx context.Context) (nodes []string, err error) {
	rd.Lock()
	defer rd.Unlock()
	sliceCmd := rd.c.ZRangeByScore(ctx, GetKeyPre(rd.serviceName), &redis.ZRangeBy{
		Min: fmt.Sprintf("%d", TimePre(time.Now(), rd.timeout)),
		Max: "+inf",
	})
	if err = sliceCmd.Err(); err != nil {
		return nil, err
	} else {
		nodes = make([]string, len(sliceCmd.Val()))
		copy(nodes, sliceCmd.Val())
	}
	rd.logger.Infof("nodes=%v", nodes)
	return
}
func (rd *RedisZSetDriver) Start(ctx context.Context) (err error) {
	rd.Lock()
	defer rd.Unlock()
	if rd.started {
		err = errors.New("this driver is started")
		return
	}
	rd.runtimeCtx, rd.runtimeCancel = context.WithCancel(context.TODO())
	rd.started = true
	// register
	err = rd.registerServiceNode()
	if err != nil {
		rd.logger.Errorf("register service error=%v", err)
		return
	}
	// heartbeat timer
	go rd.heartBeat()
	return
}
func (rd *RedisZSetDriver) Stop(ctx context.Context) (err error) {
	rd.Lock()
	defer rd.Unlock()
	rd.runtimeCancel()
	rd.started = false
	return
}

func (rd *RedisZSetDriver) withOption(opt Option) (err error) {
	switch opt.Type() {
	case OptionTypeTimeout:
		{
			rd.timeout = opt.(TimeoutOption).timeout
		}
	case OptionTypeLogger:
		{
			rd.logger = opt.(LoggerOption).logger
		}
	}
	return
}

// private function

func (rd *RedisZSetDriver) heartBeat() {
	tick := time.NewTicker(rd.timeout / 2)
	for {
		select {
		case <-tick.C:
			{
				if err := rd.registerServiceNode(); err != nil {
					rd.logger.Errorf("register service node error %+v", err)
				}
			}
		case <-rd.runtimeCtx.Done():
			{
				if err := rd.c.Del(context.Background(), rd.nodeID, rd.nodeID).Err(); err != nil {
					rd.logger.Errorf("unregister service node error %+v", err)
				}
				return
			}
		}
	}
}

func (rd *RedisZSetDriver) registerServiceNode() error {
	return rd.c.ZAdd(context.Background(), GetKeyPre(rd.serviceName), &redis.Z{
		Score:  float64(time.Now().Unix()),
		Member: rd.nodeID,
	}).Err()
}
