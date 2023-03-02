package redis

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/libi/dcron/dlog"
	"github.com/libi/dcron/driver"
)

// RedisDriver is redisDriver
type RedisDriver struct {
	client  *redis.Client
	timeout time.Duration
	Key     string
	logger  dlog.Logger
}

// NewDriver return a redis driver
func NewDriver(opts *redis.Options) (*RedisDriver, error) {
	return &RedisDriver{
		client: redis.NewClient(opts),
		logger: &dlog.StdLogger{
			Log: log.Default(),
		},
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
			rd.logger.Errorf("redis expire error %+v", err)
			continue
		}
		if !keyExist {
			if err := rd.registerServiceNode(nodeID); err != nil {
				rd.logger.Errorf("register service node error %+v", err)
			}
		}
	}
}

func (rd *RedisDriver) SetLogger(log dlog.Logger) {
	rd.logger = log
}

//GetServiceNodeList get a serveice node  list
func (rd *RedisDriver) GetServiceNodeList(serviceName string) ([]string, error) {
	mathStr := fmt.Sprintf("%s*", driver.GetKeyPre(serviceName))
	return rd.scan(mathStr)
}

//RegisterServiceNode  register a service node
func (rd *RedisDriver) RegisterServiceNode(serviceName string) (nodeID string, err error) {
	nodeID = driver.GetNodeId(serviceName)
	if err := rd.registerServiceNode(nodeID); err != nil {
		return "", err
	}
	return nodeID, nil
}

func (rd *RedisDriver) registerServiceNode(nodeID string) error {
	return rd.client.SetEX(context.Background(), nodeID, nodeID, rd.timeout).Err()
}

func (rd *RedisDriver) scan(matchStr string) ([]string, error) {
	ret := make([]string, 0)
	ctx := context.Background()
	iter := rd.client.Scan(ctx, 0, matchStr, -1).Iterator()
	for iter.Next(ctx) {
		err := iter.Err()
		if err != nil {
			return nil, err
		}
		ret = append(ret, iter.Val())
	}
	return ret, nil
}

/**
Use redis transaction to make the store / remove safety.
**/

func (rd *RedisDriver) SupportStableJob() bool { return true }

func (rd *RedisDriver) Store(serviceName string, key string, body []byte) (err error) {
	storeName := driver.GetStableJobStore(serviceName)
	ctx := context.Background()
	txKey := driver.GetStableJobStoreTxKey(serviceName)
	return rd.client.Watch(ctx, func(tx *redis.Tx) error {
		// update the txKey first.
		if errInner := tx.Set(ctx, txKey, uuid.New().String(), 0).Err(); errInner != nil {
			rd.logger.Errorf("Store Incr: %v", errInner)
			return errInner
		}

		// check if job is existed
		if existBody, errInner := tx.HGet(ctx, storeName, key).Result(); existBody != "" && errInner == nil {
			rd.logger.Errorf("this job is existed, %s", key)
			return errors.New("job existed")
		}

		// set job info into it
		if errInner := tx.HSet(ctx, storeName, key, body).Err(); errInner != nil {
			rd.logger.Errorf("Store body to server: %v", errInner)
			return errInner
		}
		return nil
	}, txKey)
}

func (rd *RedisDriver) Get(serviceName string, key string) (body []byte, err error) {
	storeName := driver.GetStableJobStore(serviceName)
	existBody, err := rd.client.HGet(context.Background(), storeName, key).Result()
	if err != nil {
		return nil, err
	}
	return []byte(existBody), nil
}

func (rd *RedisDriver) Remove(serviceName string, key string) (err error) {
	storeName := driver.GetStableJobStore(serviceName)
	ctx := context.Background()
	txKey := driver.GetStableJobStoreTxKey(serviceName)
	return rd.client.Watch(ctx, func(tx *redis.Tx) error {
		if errInner := tx.Incr(ctx, txKey).Err(); errInner != nil {
			rd.logger.Errorf("Remove Incr: %v", errInner)
			return errInner
		}
		if errInner := tx.HDel(ctx, storeName, key).Err(); errInner != nil {
			rd.logger.Errorf("Remove from server: %v", errInner)
			return errInner
		}
		return nil
	}, txKey)
}
