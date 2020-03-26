package dcron

import (
	"github.com/dlvkin/dcron/consistenthash"
	"github.com/dlvkin/dcron/driver"
	"log"
	"sync"
	"time"
)

const defaultReplicas = 50
const defaultIntervalDuration = 45 * time.Second

//NodePool is a node pool
type NodePool struct {
	serviceName string
	NodeID      string
	interval     time.Duration
	mu    sync.Mutex
	nodes *consistenthash.Map

	Driver driver.Driver
	opts   PoolOptions

	dcron *Dcron
}

//PoolOptions is a pool options
type PoolOptions struct {
	Replicas int
	HashFn   consistenthash.Hash
}

func newNodePool(serverName string, driver driver.Driver, dcron *Dcron) *NodePool {

	nodePool := new(NodePool)
	nodePool.interval = defaultIntervalDuration
	nodePool.Driver = driver
	err := nodePool.Driver.Ping()
	if err != nil {
		log.Printf("newNodePool connect fail %s",err)
	}
	nodePool.serviceName = serverName
	nodePool.dcron = dcron
	option := PoolOptions{
		Replicas: defaultReplicas,
	}
	nodePool.opts = option
	return nodePool
}

func (np *NodePool) InitPoolGrabService() {
	np.NodeID = np.Driver.RegisterServiceNode(np.serviceName,np.interval)
	go np.Driver.DoHeartBeat(np.NodeID,np.interval)
}

func (np *NodePool) updatePool() {
	np.mu.Lock()
	defer np.mu.Unlock()
	nodes, err := np.Driver.GetServiceNodeList(np.serviceName)
	if nodes == nil {
		go np.InitPoolGrabService()
		return
	} else if err != nil {
		log.Printf("update redis Pool lock fail %s",err)
		return
	} else if len(nodes) == 0 {
		go np.InitPoolGrabService()
		return
	}
	np.nodes = consistenthash.New(np.opts.Replicas, np.opts.HashFn)
	for _, node := range nodes {
		np.nodes.Add(node)
	}
}

func (np *NodePool) tickerUpdatePool() {
	tickers := time.NewTicker(np.interval)
	for range tickers.C {
		if np.dcron.isRun {
			np.updatePool()
		}
	}
}
//PickNodeByJobName : 使用一致性hash算法根据任务名获取一个执行节点
func (np *NodePool) PickNodeByJobName(jobName string) string {
	np.mu.Lock()
	defer np.mu.Unlock()
	if np.nodes == nil || np.nodes.IsEmpty() {
		return ""
	}
	return np.nodes.Get(jobName)
}

// SetInterval set interval time
func (np *NodePool) SetInterval(interval time.Duration)  {
	np.mu.Lock()
	defer np.mu.Unlock()
	if interval > 5 *time.Second {
		np.interval = interval
	} else {
		np.interval = defaultIntervalDuration
	}
}