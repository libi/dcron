package driver

import (
	"time"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"github.com/satori/go.uuid"
)

const GlobalKeyPrefix  = "distributed-cron:"

type RedisDriver struct {
	redisClient *redis.Pool
	timeout time.Duration
	Key string
	serverName string
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

func(this *RedisDriver)getKeyPre() string{
	return  GlobalKeyPrefix+this.serverName+":";

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

func(this *RedisDriver)GetNodeList(serverName string) []string{
	nodes := make([]string,0)


	keys,err := redis.ByteSlices(this.do("keys",this.getKeyPre()+"*"))
	if(err != nil){
		return nodes
	}
	for _,key := range keys{
		nodes = append(nodes,string(key))
	}
	return nodes
}
func(this *RedisDriver)RegisterNode(serverName string) (nodeId string){

	this.serverName = serverName

	nodeId = uuid.Must(uuid.NewV4()).String()

	key := this.getKeyPre()+nodeId
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

func init(){
	RegisterDriver("redis",new(RedisDriver))
}


