package dcron

import (
	"log"
	"sync"
	"time"

	"github.com/libi/dcron/consistenthash"
	"github.com/libi/dcron/dlog"
	"github.com/libi/dcron/driver"
)

// NodePool is a node pool
type NodePool struct {
	serviceName string
	NodeID      string

	rwMut sync.RWMutex
	nodes *consistenthash.Map

	driver         driver.DriverV2
	hashReplicas   int
	hashFn         consistenthash.Hash
	updateDuration time.Duration

	logger    dlog.Logger
	closeChan chan interface{}
	nodesChan chan []string
}

func newNodePool(serviceName string, driver driver.DriverV2, updateDuration time.Duration, hashReplicas int, logger dlog.Logger) *NodePool {
	np := &NodePool{
		serviceName:    serviceName,
		driver:         driver,
		hashReplicas:   hashReplicas,
		updateDuration: updateDuration,
		logger: &dlog.StdLogger{
			Log: log.Default(),
		},
	}
	if logger != nil {
		np.logger = logger
	}
	np.NodeID = np.driver.Init(serviceName, updateDuration, np.logger)
	return np
}

func (np *NodePool) StartPool() (err error) {
	np.nodesChan, err = np.driver.Start()
	go np.waitingForHashRing()
	return
}

// Check if this job can be run in this node.
func (np *NodePool) CheckJobAvailable(jobName string) bool {
	np.rwMut.RLock()
	defer np.rwMut.RUnlock()
	if np.nodes == nil {
		np.logger.Errorf("nodeID=%s, np.nodes is nil", np.NodeID)
	}
	if np.nodes.IsEmpty() {
		return false
	}
	targetNode := np.nodes.Get(jobName)
	if np.NodeID == targetNode {
		np.logger.Infof("job %s, running in node: %s", jobName, targetNode)
	}
	return np.NodeID == targetNode
}

func (np *NodePool) Close() {
	close(np.closeChan)
}

func (np *NodePool) waitingForHashRing() {
	for {
		select {
		case nowNodes := <-np.nodesChan:
			np.logger.Infof("update hashRing nowNodes=%+v", nowNodes)
			np.updateHashRing(nowNodes)
		case <-np.closeChan:
			return
		}
	}
}

func (np *NodePool) updateHashRing(nodes []string) {
	np.rwMut.Lock()
	defer np.rwMut.Unlock()
	np.nodes = consistenthash.New(np.hashReplicas, np.hashFn)
	for _, v := range nodes {
		np.nodes.Add(v)
	}
}
