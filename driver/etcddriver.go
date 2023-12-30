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

	cli    *clientv3.Client
	nodes  *sync.Map
	logger dlog.Logger

	leaseTimeout int64
	leaseID      clientv3.LeaseID
	lease        clientv3.Lease

	ctx    context.Context
	cancel context.CancelFunc
}

// NewEtcdDriver
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

// 设置key value，绑定租约
func (e *EtcdDriver) putKeyWithLease(ctx context.Context, key, val string) (clientv3.LeaseID, error) {
	//设置租约时间，最少5s
	if e.leaseTimeout < etcdDefaultLease {
		e.leaseTimeout = etcdDefaultLease
	}
	subCtx, cancel := context.WithTimeout(ctx, etcdBusinessTimeout)
	defer cancel()
	resp, err := e.lease.Grant(ctx, e.leaseTimeout)
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
	prefix := GetKeyPre(serviceName)
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

func (e *EtcdDriver) createLease(ctx context.Context, nodeID string) (<-chan *clientv3.LeaseKeepAliveResponse, error) {
	var err error
	e.lease = clientv3.NewLease(e.cli)
	e.leaseID, err = e.putKeyWithLease(ctx, nodeID, nodeID)
	if err != nil {
		e.logger.Errorf("putKeyWithLease error: %v", err)
		return nil, err
	}
	return e.lease.KeepAlive(ctx, e.leaseID)
}

func (e *EtcdDriver) revoke(ctx context.Context) {
	_, err := e.lease.Revoke(ctx, e.leaseID)
	if err != nil {
		e.logger.Infof("lease revoke error: %v", err)
	}
}

func (e *EtcdDriver) heartBeat(ctx context.Context) {
label:
	leaseCh, err := e.createLease(ctx, e.nodeID)
	if err != nil {
		e.logger.Errorf("keep alive error, %v", err)
		return
	}
	for {
		select {
		case <-e.ctx.Done():
			{
				e.logger.Infof("driver stopped")
				return
			}
		case resp, ok := <-leaseCh:
			{
				// if lease timeout, goto top of
				// this function to keepalive
				if !ok {
					e.logger.Errorf("extend lease error, release")
					goto label
				}

				e.logger.Infof("leaseID=%0x", resp.ID)
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
	// renew a global ctx when start every time
	e.ctx, e.cancel = context.WithCancel(context.TODO())
	go e.heartBeat(ctx)
	err = e.watchService(ctx, e.serviceName)
	if err != nil {
		return
	}
	return nil
}

func (e *EtcdDriver) Stop(ctx context.Context) (err error) {
	e.revoke(ctx)
	e.cancel()
	return
}

func (e *EtcdDriver) withOption(opt Option) (err error) {
	switch opt.Type() {
	case OptionTypeTimeout:
		{
			e.leaseTimeout = int64(opt.(TimeoutOption).timeout.Seconds())
		}
	case OptionTypeLogger:
		{
			e.logger = opt.(LoggerOption).logger
		}
	}
	return
}
