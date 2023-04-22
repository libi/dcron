package driver

import (
	"github.com/go-redis/redis/v8"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// There is only one driver for one dcron.
// Tips for write a user-defined Driver by yourself.
//  1. Confirm that `Stop` and `Start` can be called for more times.
//  2. Must make `GetNodes` will return error when timeout.
type DriverV2 interface {
	// init driver
	Init(serviceName string, opts ...Option)
	// get nodeID
	NodeID() string
	// get nodes
	GetNodes() (nodes []string, err error)
	Start() (err error)
	Stop() (err error)

	withOption(opt Option) (err error)
}

func NewRedisDriver(redisClient *redis.Client) DriverV2 {
	return newRedisDriver(redisClient)
}

func NewEtcdDriver(etcdCli *clientv3.Client) DriverV2 {
	return newEtcdDriver(etcdCli)
}
