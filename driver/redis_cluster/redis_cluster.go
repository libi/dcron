package redis_cluster

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"sync"
	"time"
)

// GlobalKeyPrefix is global redis key preifx
const GlobalKeyPrefix = "distributed-cron:"

// Conf is redis cluster client config
type Conf struct {
	Proto string

	// first use addr
	Addrs    []string
	Password string

	MaxRedirects int
	ReadOnly     bool

	TLSConfig *tls.Config
}

// RedisClusterDriver is
type RedisClusterDriver struct {
	conf        *Conf
	redisClient *redis.ClusterClient
	timeout     time.Duration
	Key         string
	ctx         context.Context
}

// NewDriver return a redis driver
func NewDriver(conf *Conf) (*RedisClusterDriver, error) {
	opts := &redis.ClusterOptions{
		Addrs:    conf.Addrs,
		Password: conf.Password,
		ReadOnly: conf.ReadOnly,
	}
	if conf.MaxRedirects > 0 {
		opts.MaxRedirects = conf.MaxRedirects
	}
	if conf.TLSConfig != nil {
		opts.TLSConfig = conf.TLSConfig
	}
	redisClient := redis.NewClusterClient(opts)
	return &RedisClusterDriver{
		conf:        conf,
		redisClient: redisClient,
		ctx:         context.TODO(),
	}, nil
}

// Ping to check redis cluster is valid or not
func (rd *RedisClusterDriver) Ping() error {
	if err := rd.redisClient.Set(rd.ctx, "ping", "pong", 0).Err(); err != nil {
		return err
	}
	return nil
}
func (rd *RedisClusterDriver) getKeyPre(serviceName string) string {
	return GlobalKeyPrefix + serviceName + ":"
}

//SetTimeout set redis key expiration timeout
func (rd *RedisClusterDriver) SetTimeout(timeout time.Duration) {
	rd.timeout = timeout
}

//SetHeartBeat set heartbeat
func (rd *RedisClusterDriver) SetHeartBeat(nodeID string) {

	go rd.heartBear(nodeID)
}
func (rd *RedisClusterDriver) heartBear(nodeID string) {
	//每间隔timeout/2设置一次key的超时时间为timeout
	key := nodeID
	tickers := time.NewTicker(rd.timeout / 2)
	for range tickers.C {

		if err := rd.redisClient.Expire(rd.ctx, key, rd.timeout).Err(); err != nil {
			panic(err)
		}
	}
}

//GetServiceNodeList get a service node  list on redis cluster
func (rd *RedisClusterDriver) GetServiceNodeList(serviceName string) ([]string, error) {
	mathStr := fmt.Sprintf("%s*", rd.getKeyPre(serviceName))
	return rd.scan(mathStr)
}

//RegisterServiceNode  register a service node
func (rd *RedisClusterDriver) RegisterServiceNode(serviceName string) (nodeID string) {

	nodeID = uuid.New().String()

	key := rd.getKeyPre(serviceName) + nodeID

	if err := rd.redisClient.Set(rd.ctx, key, nodeID, rd.timeout).Err(); err != nil {
		return ""
	}
	return key
}

/**
集群模式下，scan命令只能在单机上执行，因此需要遍历master节点进行合并
*/
func (rd *RedisClusterDriver) scan(matchStr string) ([]string, error) {
	l := newSyncList()
	// scan不能直接执行，只能在每个master节点上上逐个执行再合并
	if err := rd.redisClient.ForEachMaster(rd.ctx, func(ctx context.Context, master *redis.Client) error {
		iter := master.Scan(ctx, 0, matchStr, -1).Iterator()
		for iter.Next(rd.ctx) {
			l.Append(iter.Val())
		}
		if err := iter.Err(); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return l.Values(), err
	}
	return l.Values(), nil
}

type syncList struct {
	sync.Mutex
	arr []string
}

func newSyncList() *syncList {
	l := new(syncList)
	l.arr = make([]string, 0)
	return l
}

func (l *syncList) Append(val string) {
	l.Lock()
	defer l.Unlock()
	l.arr = append(l.arr, val)
}

func (l *syncList) Values() []string {
	return l.arr
}
