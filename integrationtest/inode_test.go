package main

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/libi/dcron"
	"github.com/libi/dcron/commons"
	"github.com/libi/dcron/commons/dlog"
	"github.com/libi/dcron/consistenthash"
	"github.com/libi/dcron/driver/etcddriver"
	"github.com/libi/dcron/driver/redisdriver"
	"github.com/libi/dcron/driver/rediszsetdriver"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/suite"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/tests/v3/integration"
)

type Te struct{}
type testINodePoolSuite struct {
	suite.Suite

	rds                 *miniredis.Miniredis
	etcdsvr             integration.LazyCluster
	defaultHashReplicas int
}

func (ts *testINodePoolSuite) SetupTest() {
	ts.defaultHashReplicas = 10
}

func (ts *testINodePoolSuite) TearDownTest() {
	if ts.rds != nil {
		ts.rds.Close()
		ts.rds = nil
	}
	if ts.etcdsvr != nil {
		ts.etcdsvr.Terminate()
		ts.etcdsvr = nil
	}
}

func (ts *testINodePoolSuite) setUpRedis() {
	ts.rds = miniredis.RunT(ts.T())
}

func (ts *testINodePoolSuite) setUpEtcd() {
	ts.etcdsvr = integration.NewLazyCluster()
}

func (ts *testINodePoolSuite) stopAllNodePools(nodePools []dcron.INodePool) {
	for _, nodePool := range nodePools {
		nodePool.Stop(context.Background())
	}
}

func (ts *testINodePoolSuite) declareRedisDrivers(clients *[]*redis.Client, drivers *[]commons.DriverV2, numberOfNodes int) {
	for i := 0; i < numberOfNodes; i++ {
		*clients = append(*clients, redis.NewClient(&redis.Options{
			Addr: ts.rds.Addr(),
		}))
		*drivers = append(*drivers, redisdriver.NewDriver((*clients)[i]))
	}
}

func (ts *testINodePoolSuite) declareEtcdDrivers(clients *[]*clientv3.Client, drivers *[]commons.DriverV2, numberOfNodes int) {
	for i := 0; i < numberOfNodes; i++ {
		cli, err := clientv3.New(clientv3.Config{
			Endpoints: ts.etcdsvr.EndpointsV3(),
		})
		if err != nil {
			ts.T().Fatal(err)
		}
		*clients = append(*clients, cli)
		*drivers = append(*drivers, etcddriver.NewDriver((*clients)[i]))
	}
}

func (ts *testINodePoolSuite) declareRedisZSetDrivers(clients *[]*redis.Client, drivers *[]commons.DriverV2, numberOfNodes int) {
	for i := 0; i < numberOfNodes; i++ {
		*clients = append(*clients, redis.NewClient(&redis.Options{
			Addr: ts.rds.Addr(),
		}))
		*drivers = append(*drivers, rediszsetdriver.NewDriver((*clients)[i]))
	}
}

func (ts *testINodePoolSuite) runCheckJobAvailable(numberOfNodes int, nodePools *[]dcron.INodePool, updateDuration time.Duration) {
	for i := 0; i < numberOfNodes; i++ {
		err := (*nodePools)[i].Start(context.Background())
		ts.Require().Nil(err)
	}
	<-time.After(updateDuration * 2)
	ring := consistenthash.New(ts.defaultHashReplicas, nil)
	for _, v := range *nodePools {
		ring.Add(v.GetNodeID())
	}

	for i := 0; i < 10000; i++ {
		for j := 0; j < numberOfNodes; j++ {
			ok, err := (*nodePools)[j].CheckJobAvailable(strconv.Itoa(i))
			ts.Require().Nil(err)
			ts.Require().Equal(
				ok,
				(ring.Get(strconv.Itoa(i)) == (*nodePools)[j].GetNodeID()),
			)
		}
	}
}

func (ts *testINodePoolSuite) TestMultiNodesRedis() {
	var clients []*redis.Client
	var drivers []commons.DriverV2
	var nodePools []dcron.INodePool

	numberOfNodes := 5
	ServiceName := "TestMultiNodesRedis"
	updateDuration := 2 * time.Second
	ts.setUpRedis()
	ts.declareRedisDrivers(&clients, &drivers, numberOfNodes)

	for i := 0; i < numberOfNodes; i++ {
		nodePools = append(nodePools, dcron.NewNodePool(ServiceName, drivers[i], updateDuration, ts.defaultHashReplicas, nil))
	}
	ts.runCheckJobAvailable(numberOfNodes, &nodePools, updateDuration)
	ts.stopAllNodePools(nodePools)
}

func (ts *testINodePoolSuite) TestMultiNodesEtcd() {
	var clients []*clientv3.Client
	var drivers []commons.DriverV2
	var nodePools []dcron.INodePool

	numberOfNodes := 5
	ServiceName := "TestMultiNodesEtcd"
	updateDuration := 8 * time.Second

	ts.setUpEtcd()
	ts.declareEtcdDrivers(&clients, &drivers, numberOfNodes)

	for i := 0; i < numberOfNodes; i++ {
		nodePools = append(nodePools, dcron.NewNodePool(ServiceName, drivers[i], updateDuration, ts.defaultHashReplicas, nil))
	}
	ts.runCheckJobAvailable(numberOfNodes, &nodePools, updateDuration)
	ts.stopAllNodePools(nodePools)
}

func (ts *testINodePoolSuite) TestMultiNodesRedisZSet() {
	var clients []*redis.Client
	var drivers []commons.DriverV2
	var nodePools []dcron.INodePool

	numberOfNodes := 5
	ServiceName := "TestMultiNodesZSet"
	updateDuration := 2 * time.Second

	ts.setUpRedis()
	ts.declareRedisZSetDrivers(&clients, &drivers, numberOfNodes)

	for i := 0; i < numberOfNodes; i++ {
		nodePools = append(nodePools, dcron.NewNodePool(ServiceName, drivers[i], updateDuration, ts.defaultHashReplicas, nil))
	}
	ts.runCheckJobAvailable(numberOfNodes, &nodePools, updateDuration)
	ts.stopAllNodePools(nodePools)
}

func (ts *testINodePoolSuite) TestCheckJobAvailableFailedWithNodePoolRingIsNil() {
	np := &dcron.NodePool{}
	np.SetLogger(dlog.NewLoggerForTest(ts.T()))
	_, err := np.CheckJobAvailable("testjob")
	ts.Equal(dcron.ErrNodePoolIsNil, err)
}

func TestINodePoolTestSuite(t *testing.T) {
	s := new(testINodePoolSuite)
	suite.Run(t, s)
}
