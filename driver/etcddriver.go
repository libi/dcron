package driver

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/libi/dcron/dlog"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
)

const (
	etcdDefaultLease    = 5 // min lease time
	etcdDialTimeout     = 3 * time.Second
	etcdBusinessTimeout = 5 * time.Second
)

type EtcdDriver struct {
	nodeID      string
	serviceName string

	cli     *clientv3.Client
	lease   int64
	nodes   *sync.Map
	leaseID clientv3.LeaseID
	logger  dlog.Logger

	stopChan chan int
}

//NewEtcdDriver
func newEtcdDriver(cli *clientv3.Client) *EtcdDriver {
	ser := &EtcdDriver{
		cli:   cli,
		nodes: &sync.Map{},
		logger: &dlog.StdLogger{
			Log: log.Default(),
		},
	}

	return ser
}

//设置key value，绑定租约
func (e *EtcdDriver) putKeyWithLease(key, val string) (clientv3.LeaseID, error) {
	//设置租约时间，最少5s
	if e.lease < etcdDefaultLease {
		e.lease = etcdDefaultLease
	}

	ctx, cancel := context.WithTimeout(context.Background(), etcdBusinessTimeout)
	defer cancel()
	resp, err := e.cli.Grant(ctx, e.lease)
	if err != nil {
		return 0, err
	}
	//注册服务并绑定租约
	_, err = e.cli.Put(ctx, key, val, clientv3.WithLease(resp.ID))
	if err != nil {
		return 0, err
	}

	return resp.ID, nil
}

//WatchService 初始化服务列表和监视
func (e *EtcdDriver) watchService(serviceName string) error {
	prefix := GetKeyPre(serviceName)
	// 根据前缀获取现有的key
	resp, err := e.cli.Get(context.Background(), prefix, clientv3.WithPrefix())
	if err != nil {
		return err
	}

	for _, ev := range resp.Kvs {
		e.setServiceList(string(ev.Key), string(ev.Value))
	}

	// 监视前缀，修改变更的server
	go e.watcher(serviceName)
	return nil
}

// watcher 监听前缀
func (e *EtcdDriver) watcher(serviceName string) {
	prefix := GetKeyPre(serviceName)
	rch := e.cli.Watch(context.Background(), prefix, clientv3.WithPrefix())
	for wresp := range rch {
		for _, ev := range wresp.Events {
			switch ev.Type {
			case mvccpb.PUT: //修改或者新增
				e.setServiceList(string(ev.Kv.Key), string(ev.Kv.Value))
			case mvccpb.DELETE: //删除
				e.delServiceList(string(ev.Kv.Key))
			}
		}
	}
}

// setServiceList 新增服务地址
func (e *EtcdDriver) setServiceList(key, val string) {
	e.nodes.Store(key, val)
}

// DelServiceList 删除服务地址
func (e *EtcdDriver) delServiceList(key string) {
	e.nodes.Delete(key)
}

// GetServices 获取服务地址
func (e *EtcdDriver) getServices() []string {
	addrs := make([]string, 0)
	e.nodes.Range(func(key, _ interface{}) bool {
		addrs = append(addrs, key.(string))
		return true
	})
	return addrs
}

func (e *EtcdDriver) keepAlive(ctx context.Context, nodeID string) (<-chan *clientv3.LeaseKeepAliveResponse, error) {
	var err error
	e.leaseID, err = e.putKeyWithLease(nodeID, nodeID)
	if err != nil {
		e.logger.Errorf("putKeyWithLease error: %v", err)
		return nil, err
	}

	return e.cli.KeepAlive(ctx, e.leaseID)
}

func (e *EtcdDriver) revoke() {
	_, err := e.cli.Lease.Revoke(context.Background(), e.leaseID)
	if err != nil {
		e.logger.Printf("lease revoke error: %v", err)
	}
}

func (e *EtcdDriver) heartBeat() {
label:
	leaseCh, err := e.keepAlive(context.Background(), e.nodeID)
	if err != nil {
		return
	}
	for {
		select {
		case <-e.stopChan:
			{
				e.revoke()
				e.logger.Errorf("driver stopped")
				return
			}
		case _, ok := <-leaseCh:
			{
				// if lease timeout, goto top of
				// this function to keepalive
				if !ok {
					goto label
				}
			}
		case <-time.After(etcdBusinessTimeout):
			{
				e.logger.Errorf("ectd cli keepalive timeout")
				return
			}
		case <-time.After(time.Duration(e.lease/2) * (time.Second)):
			{
				// if near to nodes time,
				// renew the lease
				goto label
			}
		}
	}
}

func (e *EtcdDriver) Init(serverName string, opts ...Option) {
	e.serviceName = serverName
	e.nodeID = GetNodeId(serverName)
}

func (e *EtcdDriver) NodeID() string {
	return e.nodeID
}

func (e *EtcdDriver) GetNodes(ctx context.Context) (nodes []string, err error) {
	return e.getServices(), nil
}

func (e *EtcdDriver) Start(ctx context.Context) (err error) {
	e.stopChan = make(chan int, 1)
	go e.heartBeat()
	err = e.watchService(e.serviceName)
	if err != nil {
		return
	}
	return nil
}

func (e *EtcdDriver) Stop(ctx context.Context) (err error) {
	close(e.stopChan)
	return
}

func (e *EtcdDriver) withOption(opt Option) (err error) {
	switch opt.Type() {
	case OptionTypeTimeout:
		{
			e.lease = int64(opt.(TimeoutOption).timeout.Seconds())
		}
	case OptionTypeLogger:
		{
			e.logger = opt.(LoggerOption).logger
		}
	}
	return
}
