package dcron

import (
	"github.com/libi/dcron/consistenthash"
	"github.com/libi/dcron/driver"
	"sync"
	"sync/atomic"
	"time"
)

// NodePool is a node pool
type NodePool struct {
	serviceName string
	NodeID      string

	mu    sync.Mutex
	nodes *consistenthash.Map

	Driver         driver.Driver
	hashReplicas   int
	hashFn         consistenthash.Hash
	updateDuration time.Duration

	dcron *Dcron
}

func newNodePool(serverName string, driver driver.Driver, dcron *Dcron, updateDuration time.Duration, hashReplicas int) (*NodePool, error) {

	err := driver.Ping()
	if err != nil {
		return nil, err
	}

	nodePool := &NodePool{
		Driver:         driver,
		serviceName:    serverName,
		dcron:          dcron,
		hashReplicas:   hashReplicas,
		updateDuration: updateDuration,
	}
	return nodePool, nil
}

// StartPool Start Service Watch Pool
func (np *NodePool) StartPool() error {
	var err error
	np.Driver.SetTimeout(np.updateDuration)
	np.NodeID, err = np.Driver.RegisterServiceNode(np.serviceName)
	if err != nil {
		return err
	}
	np.Driver.SetHeartBeat(np.NodeID)

	err = np.updatePool()
	if err != nil {
		return err
	}

	go np.tickerUpdatePool()
	return nil
}

func (np *NodePool) updatePool() error {
	nodes, err := np.Driver.GetServiceNodeList(np.serviceName)
	if err != nil {
		return err
	}

	np.mu.Lock()
	defer np.mu.Unlock()
	np.nodes = consistenthash.New(np.hashReplicas, np.hashFn)
	for _, node := range nodes {
		np.nodes.Add(node)
	}
	return nil
}
func (np *NodePool) tickerUpdatePool() {
	tickers := time.NewTicker(np.updateDuration)
	for range tickers.C {
		if atomic.LoadInt32(&np.dcron.running) == dcronRunning {
			err := np.updatePool()
			if err != nil {
				np.dcron.err("update node pool error %+v", err)
			}
		} else {
			tickers.Stop()
			return
		}
	}
}

// PickNodeByJobName : 使用一致性hash算法根据任务名获取一个执行节点
func (np *NodePool) PickNodeByJobName(jobName string) string {
	np.mu.Lock()
	defer np.mu.Unlock()
	if np.nodes.IsEmpty() {
		return ""
	}
	return np.nodes.Get(jobName)
}
