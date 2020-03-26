package redis

import (
	"errors"
	"fmt"
	"github.com/gomodule/redigo/redis"
	"github.com/google/uuid"
	"log"
	"time"
)

// GlobalKeyPrefix is global redis key preifx
const GlobalKeyPrefix = "distributed-cron:"

// RedisConf is redis config
type Conf struct {
	Host     string
	Port     int
	Password string
}

// RedisDriver is redisDriver
type RedisDriver struct {
	conf        *Conf
	redisClient *redis.Pool
	Key         string
	alive    bool
}

// NewDriver return a redis driver
func NewDriver(conf *Conf) (*RedisDriver, error) {
	rd := &redis.Pool{
		MaxIdle:     100,
		MaxActive:   100,
		IdleTimeout: 5 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", fmt.Sprintf("%s:%d", conf.Host, conf.Port),
				redis.DialConnectTimeout(time.Second*5), redis.DialPassword(conf.Password))
			if err != nil {
				log.Printf("fail redis connect  %s",err)
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
	defer func(){
		if err := recover(); err != nil {
			log.Printf("fail redis pool ping %v", err)
		}
	}()
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

//DoHeartBeat set heart beat
func (rd *RedisDriver) DoHeartBeat(nodeID string,timeout time.Duration) {
	//每间隔timeout/2设置一次key的超时时间为timeout
	if rd.alive {
		return
	}
	tickers := time.NewTicker(timeout/2)
	rd.alive = true
	key := nodeID
	close := func (){
		rd.alive = false
		tickers.Stop()
	}
	defer close()
	for range tickers.C {
		_, err := rd.do("EXPIRE",key, int(timeout/time.Second))
		if err != nil {
			return
		}
	}
}

func (rd *RedisDriver) IsCheckAlive() bool {
	return rd.alive
}

//GetServiceNodeList get a serveice node  list
func (rd *RedisDriver) GetServiceNodeList(serviceName string) ([]string, error) {
	mathStr := fmt.Sprintf("%s*", rd.getKeyPre(serviceName))
	return rd.scan(mathStr)
}

//RegisterServiceNode  register a service node
func (rd *RedisDriver) RegisterServiceNode(serviceName string,lifeTime time.Duration) (nodeID string) {
	nodeID = uuid.New().String()
	key := rd.getKeyPre(serviceName) + nodeID
	_, err := rd.do("SETEX", key, int(lifeTime/time.Second), nodeID)
	if err != nil {
		return ""
	}
	return key
}

func (rd *RedisDriver) do(command string, params ...interface{}) (interface{}, error) {
	conn := rd.redisClient.Get()
	defer func(){
		if err := recover(); err != nil {
			log.Printf(" redis do command fail: %v", err)
		}
	}()
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
