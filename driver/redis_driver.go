package driver

import (
	"errors"
	"time"
	"fmt"
	"github.com/gomodule/redigo/redis"
	"github.com/satori/go.uuid"
)

const GlobalKeyPrefix  = "distributed-cron:"

type RedisDriver struct {
	redisClient *redis.Pool
	timeout time.Duration
	Key string
}
func(this *RedisDriver)Open(dataSourceOption DriverConnOpt){

	this.redisClient = &redis.Pool{
		MaxIdle:     100,
		MaxActive:   100,
		IdleTimeout: 5 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", fmt.Sprintf("%s:%s",dataSourceOption.Host,dataSourceOption.Port),
				redis.DialConnectTimeout(time.Second*5),redis.DialPassword(dataSourceOption.Password))
			if err != nil {
				panic(err)
			}
			return c, nil
		},
	}

}

func(this *RedisDriver)getKeyPre(serviceName string) string{
	return  GlobalKeyPrefix+serviceName+":";
}

func(this *RedisDriver)SetTimeout(timeout time.Duration){
	this.timeout = timeout
}

func(this *RedisDriver)SetHeartBeat(nodeId string){

	go this.heartBear(nodeId)
}
func(this *RedisDriver)heartBear(nodeId string){

	//每间隔timeout/2设置一次key的超时时间为timeout
	key := nodeId
	tickers := time.NewTicker(this.timeout/2)
	for range tickers.C {
		_,err := this.do("EXPIRE",key,int(this.timeout/time.Second))
		if(err != nil){
			panic(err)
		}
	}
}

func(this *RedisDriver)GetServiceNodeList(serviceName string) ([]string,error){
	mathStr := fmt.Sprintf("%s*",this.getKeyPre(serviceName))
	return this.scan(mathStr)
}
func(this *RedisDriver)RegisterServiceNode(serviceName string) (nodeId string){

	nodeId = uuid.NewV4().String()

	key := this.getKeyPre(serviceName)+nodeId
	_,err := this.do("SETEX",key,int(this.timeout/time.Second),nodeId)
	if(err != nil){
		return ""
	}
	return key
}

func (this *RedisDriver)do(command string,params ...interface{}) (interface{},error){
	conn := this.redisClient.Get()
	defer conn.Close()
	return conn.Do(command,params...)
}
func (this *RedisDriver)scan(matchStr string) ([]string,error) {
	cursor := "0"
	ret := make([]string,0)
	for {
		reply ,err := this.do("scan",cursor,"match",matchStr)
		if(err != nil){
			return nil,err
		}
		if Reply,ok := reply.([]interface{});ok && len(Reply)==2 {
			cursor = string(Reply[0].([]byte))

			list := Reply[1].([]interface{})
			for _,item := range list{
				ret = append(ret,string(item.([]byte)))
			}
			if(cursor == "0"){
				break
			}
		}else{
			return nil,errors.New("redis scan resp struct error")
		}
	}
	return ret,nil
}
func init(){
	RegisterDriver("redis",new(RedisDriver))
}


