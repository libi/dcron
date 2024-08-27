package etcddriver

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/dcron-contrib/commons"
	"github.com/dcron-contrib/commons/dlog"
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

	cli    *clientv3.Client
	nodes  *sync.Map
	logger dlog.Logger

	lease   int64
	leaseID clientv3.LeaseID
	leaseCh <-chan *clientv3.LeaseKeepAliveResponse

	ctx    context.Context
	cancel context.CancelFunc
}

// NewEtcdDriver
func NewDriver(cli *clientv3.Client) *EtcdDriver {
	ser := &EtcdDriver{
		cli:   cli,
		nodes: &sync.Map{},
		logger: &dlog.StdLogger{
			Log: log.Default(),
		},
	}

	return ser
}

// 设置key value，绑定租约
func (e *EtcdDriver) putKeyWithLease(ctx context.Context, key, val string) (clientv3.LeaseID, error) {
	//设置租约时间，最少5s
	if e.lease < etcdDefaultLease {
		e.lease = etcdDefaultLease
	}

	subCtx, cancel := context.WithTimeout(ctx, etcdBusinessTimeout)
	defer cancel()
	resp, err := e.cli.Grant(subCtx, e.lease)
	if err != nil {
		return 0, err
	}
	//注册服务并绑定租约
	_, err = e.cli.Put(subCtx, key, val, clientv3.WithLease(resp.ID))
	if err != nil {
		return 0, err
	}

	return resp.ID, nil
}

// WatchService 初始化服务列表和监视
func (e *EtcdDriver) watchService(ctx context.Context, serviceName string) error {
	prefix := commons.GetKeyPre(serviceName)
	// 根据前缀获取现有的key
	resp, err := e.cli.Get(ctx, prefix, clientv3.WithPrefix())
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
	prefix := commons.GetKeyPre(serviceName)
	rch := e.cli.Watch(context.Background(), prefix, clientv3.WithPrefix())
	for wresp := range rch {
		for _, ev := range wresp.Events {
			switch ev.Type {
			case mvccpb.PUT:
				// 修改或者新增
				e.setServiceList(string(ev.Kv.Key), string(ev.Kv.Value))
			case mvccpb.DELETE:
				// 删除
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
	e.leaseID, err = e.putKeyWithLease(ctx, nodeID, nodeID)
	if err != nil {
		e.logger.Errorf("putKeyWithLease error: %v", err)
		return nil, err
	}

	return e.cli.KeepAlive(ctx, e.leaseID)
}

func (e *EtcdDriver) revoke(ctx context.Context) {
	_, err := e.cli.Lease.Revoke(ctx, e.leaseID)
	if err != nil {
		e.logger.Infof("lease revoke error: %v", err)
	}
}

func (e *EtcdDriver) startHeartBeat(ctx context.Context) {
	var err error
	e.leaseCh, err = e.keepAlive(ctx, e.nodeID)
	if err != nil {
		e.logger.Errorf("keep alive error, %v", err)
		return
	}
}

func (e *EtcdDriver) keepHeartBeat() {
	for {
		select {
		case <-e.ctx.Done():
			{
				e.logger.Warnf("driver stopped")
				return
			}
		case _, ok := <-e.leaseCh:
			{
				if !ok {
					e.logger.Warnf("lease channel stop, driver stopped")
					return
				}
			}
		}
	}
}

func (e *EtcdDriver) Init(serverName string, opts ...commons.Option) {
	e.serviceName = serverName
	e.nodeID = commons.GetNodeId(serverName)
	for _, opt := range opts {
		e.WithOption(opt)
	}
}

func (e *EtcdDriver) NodeID() string {
	return e.nodeID
}

func (e *EtcdDriver) GetNodes(ctx context.Context) (nodes []string, err error) {
	return e.getServices(), nil
}

func (e *EtcdDriver) Start(ctx context.Context) (err error) {
	// renew a global ctx when start every time
	e.ctx, e.cancel = context.WithCancel(context.TODO())
	e.startHeartBeat(ctx)
	err = e.watchService(ctx, e.serviceName)
	if err != nil {
		return
	}
	go e.keepHeartBeat()
	return
}

func (e *EtcdDriver) Stop(ctx context.Context) (err error) {
	e.revoke(ctx)
	e.cancel()
	return
}

func (e *EtcdDriver) WithOption(opt commons.Option) (err error) {
	switch opt.Type() {
	case commons.OptionTypeTimeout:
		{
			e.lease = int64(opt.(commons.TimeoutOption).Timeout.Seconds())
		}
	case commons.OptionTypeLogger:
		{
			e.logger = opt.(commons.LoggerOption).Logger
		}
	}
	return
}
