package redis

import (
	"errors"
	"fmt"
	"github.com/gomodule/redigo/redis"
	"github.com/google/uuid"
	"time"
)

// GlobalKeyPrefix is global redis key preifx
const GlobalKeyPrefix = "distributed-cron:"

// RedisConf is redis config
type Conf struct {
	Proto string

	// first use addr
	Addr     string
	Password string

	Host string
	Port int

	MaxActive   int
	MaxIdle     int
	IdleTimeout time.Duration
	Wait        bool
}

// RedisDriver is redisDriver
type RedisDriver struct {
	conf        *Conf
	redisClient *redis.Pool
	timeout     time.Duration
	Key         string
}

// NewDriver return a redis driver
func NewDriver(conf *Conf, options ...redis.DialOption) (*RedisDriver, error) {
	ops := []redis.DialOption{
		redis.DialPassword(conf.Password),
	}
	ops = append(ops, options...)

	if conf.Proto == "" {
		conf.Proto = "tcp"
	}
	if conf.MaxActive == 0 {
		conf.MaxActive = 100
	}
	if conf.MaxIdle == 0 {
		conf.MaxIdle = 100
	}
	if conf.IdleTimeout == 0 {
		conf.IdleTimeout = time.Second * 5
	}
	if conf.Addr == "" {
		conf.Addr = fmt.Sprintf("%s:%d", conf.Host, conf.Port)
	}

	rd := &redis.Pool{
		MaxIdle:     conf.MaxIdle,
		MaxActive:   conf.MaxActive,
		IdleTimeout: conf.IdleTimeout,
		Wait:        conf.Wait,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial(conf.Proto, conf.Addr, ops...)
			if err != nil {
				panic(err)
			}
			return c, nil
		},
	}
	return &RedisDriver{
		conf:        conf,
		redisClient: rd,
	}, nil
}

// Ping is check redis valid
func (rd *RedisDriver) Ping() error {
	conn := rd.redisClient.Get()
	defer conn.Close()
	if _, err := conn.Do("SET", "ping", "pong"); err != nil {
		return err
	}
	return nil
}
func (rd *RedisDriver) getKeyPre(serviceName string) string {
	return GlobalKeyPrefix + serviceName + ":"
}

//SetTimeout set redis timeout
func (rd *RedisDriver) SetTimeout(timeout time.Duration) {
	rd.timeout = timeout
}

//SetHeartBeat set herbear
func (rd *RedisDriver) SetHeartBeat(nodeID string) {

	go rd.heartBear(nodeID)
}
func (rd *RedisDriver) heartBear(nodeID string) {

	//每间隔timeout/2设置一次key的超时时间为timeout
	key := nodeID
	tickers := time.NewTicker(rd.timeout / 2)
	for range tickers.C {
		_, err := rd.do("EXPIRE", key, int(rd.timeout/time.Second))
		if err != nil {
			panic(err)
		}
	}
}

//GetServiceNodeList get a serveice node  list
func (rd *RedisDriver) GetServiceNodeList(serviceName string) ([]string, error) {
	mathStr := fmt.Sprintf("%s*", rd.getKeyPre(serviceName))
	return rd.scan(mathStr)
}

//RegisterServiceNode  register a service node
func (rd *RedisDriver) RegisterServiceNode(serviceName string) (nodeID string) {

	nodeID = uuid.New().String()

	key := rd.getKeyPre(serviceName) + nodeID
	_, err := rd.do("SETEX", key, int(rd.timeout/time.Second), nodeID)
	if err != nil {
		return ""
	}
	return key
}

func (rd *RedisDriver) do(command string, params ...interface{}) (interface{}, error) {
	conn := rd.redisClient.Get()
	defer conn.Close()
	return conn.Do(command, params...)
}
func (rd *RedisDriver) scan(matchStr string) ([]string, error) {
	cursor := "0"
	ret := make([]string, 0)
	for {
		reply, err := rd.do("scan", cursor, "match", matchStr)
		if err != nil {
			return nil, err
		}
		if Reply, ok := reply.([]interface{}); ok && len(Reply) == 2 {
			cursor = string(Reply[0].([]byte))

			list := Reply[1].([]interface{})
			for _, item := range list {
				ret = append(ret, string(item.([]byte)))
			}
			if cursor == "0" {
				break
			}
		} else {
			return nil, errors.New("redis scan resp struct error")
		}
	}
	return ret, nil
}
