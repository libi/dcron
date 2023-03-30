package dcron

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/libi/dcron/consistenthash"
	"github.com/libi/dcron/driver"
)

// NodePool is a node pool
type NodePool struct {
	serviceName string
	NodeID      string

	rwMut sync.RWMutex
	nodes *consistenthash.Map

	Driver         driver.Driver
	hashReplicas   int
	hashFn         consistenthash.Hash
	updateDuration time.Duration

	dcron *Dcron
}

func newNodePool(serverName string, driver driver.Driver, dcron *Dcron, updateDuration time.Duration, hashReplicas int) (*NodePool, error) {
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

	np.rwMut.Lock()
	defer np.rwMut.Unlock()
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
				np.dcron.logger.Infof("update node pool error %+v", err)
			}
		} else {
			tickers.Stop()
			return
		}
	}
}

// PickNodeByJobName : 使用一致性hash算法根据任务名获取一个执行节点
func (np *NodePool) PickNodeByJobName(jobName string) string {
	np.rwMut.RLock()
	defer np.rwMut.RUnlock()
	if np.nodes.IsEmpty() {
		return ""
	}
	return np.nodes.Get(jobName)
}
