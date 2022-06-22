package etcd

import (
	"context"
	"github.com/google/uuid"
	"github.com/libi/dcron/driver"
	"go.etcd.io/etcd/api/v3/mvccpb"
	"go.etcd.io/etcd/client/v3"
	"log"
	"sync"
	"time"
)

var _ driver.Driver = &EtcdDriver{}

const (
	defaultLease    = 5 // 5 second ttl
	dialTimeout     = 3 * time.Second
	businessTimeout = 5 * time.Second
)

type EtcdDriver struct {
	cli           *clientv3.Client
	lease         int64
	leaseID       clientv3.LeaseID
	keepAliveChan <-chan *clientv3.LeaseKeepAliveResponse
	serverList    map[string]map[string]string
	lock          sync.Mutex
}

//NewEtcdDriver ...
func NewEtcdDriver(endpoints []string) (*EtcdDriver, error) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: dialTimeout,
	})
	if err != nil {
		log.Printf("NewEtcdDriver error: %v", err)
		return nil, err
	}

	ser := &EtcdDriver{
		cli:        cli,
		serverList: make(map[string]map[string]string, 10),
	}

	return ser, nil
}

//设置租约
func (s *EtcdDriver) putKeyWithLease(key, val string) error {
	//设置租约时间，最少5s
	if s.lease < defaultLease {
		s.lease = defaultLease
	}

	ctx, cancel := context.WithTimeout(context.Background(), businessTimeout)
	defer cancel()

	resp, err := s.cli.Grant(ctx, s.lease)
	if err != nil {
		log.Printf("grant error:%v", err)
		return err
	}
	//注册服务并绑定租约
	_, err = s.cli.Put(ctx, key, val, clientv3.WithLease(resp.ID))
	if err != nil {
		log.Printf("put error:%v", err)
		return err
	}
	//设置续租 定期发送需求请求 此处ctx不能设置超时
	leaseRespChan, err := s.cli.KeepAlive(context.Background(), resp.ID)

	if err != nil {
		log.Printf("keepalive error:%v", err)
		return err
	}
	s.leaseID = resp.ID
	s.keepAliveChan = leaseRespChan
	log.Printf("putKeyWithLease key:%v  val:%v  success!", key, val)
	return nil
}

//listenLeaseRespChan 监听 续租情况
func (s *EtcdDriver) listenLeaseRespChan() {
	for leaseKeepResp := range s.keepAliveChan {
		log.Printf("续约成功  %v", leaseKeepResp)
	}
	log.Printf("关闭续租")
}

// Close 注销服务
func (s *EtcdDriver) Close() error {
	//撤销租约
	if _, err := s.cli.Revoke(context.Background(), s.leaseID); err != nil {
		return err
	}
	log.Printf("撤销租约")
	return s.cli.Close()
}

func (s *EtcdDriver) randNodeID(serviceName string) (path, nodeID string) {
	nodeID = uuid.New().String()
	path = getPrefix(serviceName) + nodeID
	return
}

//WatchService 初始化服务列表和监视
func (s *EtcdDriver) watchService(serviceName string) error {
	prefix := getPrefix(serviceName)
	//根据前缀获取现有的key
	resp, err := s.cli.Get(context.Background(), prefix, clientv3.WithPrefix())
	if err != nil {
		return err
	}

	for _, ev := range resp.Kvs {
		s.setServiceList(serviceName, string(ev.Key), string(ev.Value))
	}

	//监视前缀，修改变更的server
	go s.watcher(serviceName)
	return nil
}

func getPrefix(serviceName string) string {
	return serviceName + "/"
}

//watcher 监听前缀
func (s *EtcdDriver) watcher(serviceName string) {
	prefix := getPrefix(serviceName)
	rch := s.cli.Watch(context.Background(), prefix, clientv3.WithPrefix())
	log.Printf("watching prefix:%s now...", prefix)
	for wresp := range rch {
		for _, ev := range wresp.Events {
			switch ev.Type {
			case mvccpb.PUT: //修改或者新增
				s.setServiceList(serviceName, string(ev.Kv.Key), string(ev.Kv.Value))
			case mvccpb.DELETE: //删除
				s.delServiceList(serviceName, string(ev.Kv.Key))
			}
		}
	}
}

//setServiceList 新增服务地址
func (s *EtcdDriver) setServiceList(serviceName, key, val string) {
	s.lock.Lock()
	defer s.lock.Unlock()
	log.Printf("put key :%v val:%v", key, val)
	var nodeMap map[string]string
	var ok bool
	if nodeMap, ok = s.serverList[serviceName]; !ok {
		nodeMap = make(map[string]string)
		s.serverList[serviceName] = nodeMap
	}
	nodeMap[key] = val
}

//DelServiceList 删除服务地址
func (s *EtcdDriver) delServiceList(serviceName, key string) {
	s.lock.Lock()
	defer s.lock.Unlock()
	log.Printf("del serviceName:%v, key:%v", serviceName, key)
	if nodeMap, ok := s.serverList[serviceName]; ok {
		delete(nodeMap, key)
	}
}

//GetServices 获取服务地址
func (s *EtcdDriver) getServices(serviceName string) []string {
	s.lock.Lock()
	defer s.lock.Unlock()
	addrs := make([]string, 0)
	if nodeMap, ok := s.serverList[serviceName]; ok {
		for _, v := range nodeMap {
			addrs = append(addrs, v)
		}
	}
	return addrs
}

func (e *EtcdDriver) Ping() error {
	return nil
}

func (e *EtcdDriver) SetHeartBeat(nodeID string) {
	//no need
}

func (e *EtcdDriver) SetTimeout(timeout time.Duration) {

	e.lease = int64(timeout.Seconds())
}

func (e *EtcdDriver) GetServiceNodeList(serviceName string) ([]string, error) {
	return e.getServices(serviceName), nil
}

func (e *EtcdDriver) RegisterServiceNode(serviceName string) (string, error) {

	path, nodeID := e.randNodeID(serviceName)

	//申请租约设置时间keepalive
	if err := e.putKeyWithLease(path, nodeID); err != nil {
		return "", err
	}

	go e.listenLeaseRespChan()

	e.watchService(serviceName)

	return nodeID, nil
}
