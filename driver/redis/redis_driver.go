package redis

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

// GlobalKeyPrefix is global redis key preifx
const GlobalKeyPrefix = "distributed-cron:"

// RedisDriver is redisDriver
type RedisDriver struct {
	client  *redis.Client
	timeout time.Duration
	Key     string
}

// NewDriver return a redis driver
func NewDriver(opts *redis.Options) (*RedisDriver, error) {
	return &RedisDriver{
		client: redis.NewClient(opts),
	}, nil
}

// Ping is check redis valid
func (rd *RedisDriver) Ping() error {
	reply, err := rd.client.Ping(context.Background()).Result()
	if err != nil {
		return err
	}
	if reply != "PONG" {
		return fmt.Errorf("Ping received is error, %s", string(reply))
	}
	return err
}

func (rd *RedisDriver) getKeyPre(serviceName string) string {
	return fmt.Sprintf("%s%s:", GlobalKeyPrefix, serviceName)
}

//SetTimeout set redis timeout
func (rd *RedisDriver) SetTimeout(timeout time.Duration) {
	rd.timeout = timeout
}

//SetHeartBeat set heatbeat
func (rd *RedisDriver) SetHeartBeat(nodeID string) {
	go rd.heartBeat(nodeID)
}
func (rd *RedisDriver) heartBeat(nodeID string) {

	//每间隔timeout/2设置一次key的超时时间为timeout
	key := nodeID
	tickers := time.NewTicker(rd.timeout / 2)
	for range tickers.C {
		keyExist, err := rd.client.Expire(context.Background(), key, rd.timeout).Result()
		if err != nil {
			log.Printf("redis expire error %+v", err)
			continue
		}
		if !keyExist {
			if err := rd.registerServiceNode(nodeID); err != nil {
				log.Printf("register service node error %+v", err)
			}
		}
	}
}

//GetServiceNodeList get a serveice node  list
func (rd *RedisDriver) GetServiceNodeList(serviceName string) ([]string, error) {
	mathStr := fmt.Sprintf("%s*", rd.getKeyPre(serviceName))
	return rd.scan(mathStr)
}

//RegisterServiceNode  register a service node
func (rd *RedisDriver) RegisterServiceNode(serviceName string) (nodeID string, err error) {
	nodeID = rd.randNodeID(serviceName)
	if err := rd.registerServiceNode(nodeID); err != nil {
		return "", err
	}
	return nodeID, nil
}

func (rd *RedisDriver) randNodeID(serviceName string) (nodeID string) {
	return rd.getKeyPre(serviceName) + uuid.New().String()
}

func (rd *RedisDriver) registerServiceNode(nodeID string) error {
	return rd.client.SetEX(context.Background(), nodeID, nodeID, rd.timeout).Err()
}

func (rd *RedisDriver) scan(matchStr string) ([]string, error) {
	ret := make([]string, 0)
	ctx := context.Background()
	iter := rd.client.Scan(ctx, 0, matchStr, -1).Iterator()
	for iter.Next(ctx) {
		ret = append(ret, iter.Val())
	}
	return ret, nil
}
