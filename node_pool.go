package dcron

import (
	"context"
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

	logger   dlog.Logger
	stopChan chan int
	preNodes []string // sorted
}

func NewNodePool(serviceName string, drv driver.DriverV2, updateDuration time.Duration, hashReplicas int, logger dlog.Logger) *NodePool {
	np := &NodePool{
		serviceName:    serviceName,
		driver:         drv,
		hashReplicas:   hashReplicas,
		updateDuration: updateDuration,
		logger: &dlog.StdLogger{
			Log: log.Default(),
		},
		stopChan: make(chan int, 1),
	}
	if logger != nil {
		np.logger = logger
	}
	np.driver.Init(serviceName,
		driver.NewTimeoutOption(updateDuration),
		driver.NewLoggerOption(np.logger))
	return np
}

func (np *NodePool) StartPool() (err error) {
	err = np.driver.Start(context.Background())
	if err != nil {
		np.logger.Errorf("start pool error: %v", err)
		return
	}
	np.NodeID = np.driver.NodeID()
	nowNodes, err := np.driver.GetNodes(context.Background())
	if err != nil {
		np.logger.Errorf("get nodes error: %v", err)
		return
	}
	np.updateHashRing(nowNodes)
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

func (np *NodePool) Stop() {
	np.stopChan <- 1
	np.driver.Stop(context.Background())
	np.preNodes = make([]string, 0)
}

func (np *NodePool) waitingForHashRing() {
	tick := time.NewTicker(np.updateDuration)
	for {
		select {
		case <-tick.C:
			nowNodes, err := np.driver.GetNodes(context.Background())
			if err != nil {
				np.logger.Errorf("get nodes error %v", err)
				continue
			}
			np.updateHashRing(nowNodes)
		case <-np.stopChan:
			return
		}
	}
}

func (np *NodePool) updateHashRing(nodes []string) {
	np.rwMut.Lock()
	defer np.rwMut.Unlock()
	if np.equalRing(nodes) {
		np.logger.Infof("nowNodes=%v, preNodes=%v", nodes, np.preNodes)
		return
	}
	np.logger.Infof("update hashRing nodes=%+v", nodes)
	np.preNodes = make([]string, len(nodes))
	copy(np.preNodes, nodes)
	np.nodes = consistenthash.New(np.hashReplicas, np.hashFn)
	for _, v := range nodes {
		np.nodes.Add(v)
	}
}

func (np *NodePool) equalRing(a []string) bool {
	if len(a) == len(np.preNodes) {
		la := len(a)
		for i := 0; i < la; i++ {
			if a[i] != np.preNodes[i] {
				return false
			}
		}
		return true
	}
	return false
}
