package dcron

import (
	"github.com/LibiChai/dcron/consistenthash"
	"github.com/LibiChai/dcron/driver"
	"sync"
	"time"
)

const defaultReplicas = 50
const defaultDuration = 1

//NodePool is a node pool
type NodePool struct {
	serviceName string
	NodeID      string

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
	nodePool.Driver = driver
	err := nodePool.Driver.Ping()
	if err != nil {
		panic(err)
	}

	nodePool.serviceName = serverName
	nodePool.dcron = dcron

	option := PoolOptions{
		Replicas: defaultReplicas,
	}
	nodePool.opts = option

	nodePool.initPool()
	nodePool.updatePool()

	go nodePool.tickerUpdatePool()

	return nodePool
}

func (np *NodePool) initPool() {
	np.Driver.SetTimeout(defaultDuration * time.Second)
	np.NodeID = np.Driver.RegisterServiceNode(np.serviceName)
	np.Driver.SetHeartBeat(np.NodeID)
}

func (np *NodePool) updatePool() {
	np.mu.Lock()
	defer np.mu.Unlock()
	nodes, err := np.Driver.GetServiceNodeList(np.serviceName)
	if nodes == nil {
		panic(err)
	}
	np.nodes = consistenthash.New(np.opts.Replicas, np.opts.HashFn)
	for _, node := range nodes {
		np.nodes.Add(node)
	}
}
func (np *NodePool) tickerUpdatePool() {
	tickers := time.NewTicker(time.Second * defaultDuration)
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
	if np.nodes.IsEmpty() {
		return ""
	}
	return np.nodes.Get(jobName)
}
