package dcron_test

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/libi/dcron"
	"github.com/libi/dcron/consistenthash"
	"github.com/libi/dcron/dlog"
	"github.com/libi/dcron/driver"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/suite"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/tests/v3/integration"
)

type TestINodePoolSuite struct {
	suite.Suite

	rds                 *miniredis.Miniredis
	etcdsvr             integration.LazyCluster
	defaultHashReplicas int
}

func (ts *TestINodePoolSuite) SetupTest() {
	ts.defaultHashReplicas = 10
}

func (ts *TestINodePoolSuite) TearDownTest() {
	if ts.rds != nil {
		ts.rds.Close()
		ts.rds = nil
	}
	if ts.etcdsvr != nil {
		ts.etcdsvr.Terminate()
		ts.etcdsvr = nil
	}
}

func (ts *TestINodePoolSuite) setUpRedis() {
	ts.rds = miniredis.RunT(ts.T())
}

func (ts *TestINodePoolSuite) setUpEtcd() {
	ts.etcdsvr = integration.NewLazyCluster()
}

func (ts *TestINodePoolSuite) stopAllNodePools(nodePools []dcron.INodePool) {
	for _, nodePool := range nodePools {
		nodePool.Stop(context.Background())
	}
}

func (ts *TestINodePoolSuite) declareRedisDrivers(clients *[]*redis.Client, drivers *[]driver.DriverV2, numberOfNodes int) {
	for i := 0; i < numberOfNodes; i++ {
		*clients = append(*clients, redis.NewClient(&redis.Options{
			Addr: ts.rds.Addr(),
		}))
		*drivers = append(*drivers, driver.NewRedisDriver((*clients)[i]))
	}
}

func (ts *TestINodePoolSuite) declareEtcdDrivers(clients *[]*clientv3.Client, drivers *[]driver.DriverV2, numberOfNodes int) {
	for i := 0; i < numberOfNodes; i++ {
		cli, err := clientv3.New(clientv3.Config{
			Endpoints: ts.etcdsvr.EndpointsV3(),
		})
		if err != nil {
			ts.T().Fatal(err)
		}
		*clients = append(*clients, cli)
		*drivers = append(*drivers, driver.NewEtcdDriver((*clients)[i]))
	}
}

func (ts *TestINodePoolSuite) declareRedisZSetDrivers(clients *[]*redis.Client, drivers *[]driver.DriverV2, numberOfNodes int) {
	for i := 0; i < numberOfNodes; i++ {
		*clients = append(*clients, redis.NewClient(&redis.Options{
			Addr: ts.rds.Addr(),
		}))
		*drivers = append(*drivers, driver.NewRedisZSetDriver((*clients)[i]))
	}
}

func (ts *TestINodePoolSuite) runCheckJobAvailable(numberOfNodes int, ServiceName string, nodePools *[]dcron.INodePool, updateDuration time.Duration) {
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

func (ts *TestINodePoolSuite) TestMultiNodesRedis() {
	var clients []*redis.Client
	var drivers []driver.DriverV2
	var nodePools []dcron.INodePool

	numberOfNodes := 5
	ServiceName := "TestMultiNodesRedis"
	updateDuration := 2 * time.Second
	ts.setUpRedis()
	ts.declareRedisDrivers(&clients, &drivers, numberOfNodes)

	for i := 0; i < numberOfNodes; i++ {
		nodePools = append(nodePools, dcron.NewNodePool(ServiceName, drivers[i], updateDuration, ts.defaultHashReplicas, nil))
	}
	ts.runCheckJobAvailable(numberOfNodes, ServiceName, &nodePools, updateDuration)
	ts.stopAllNodePools(nodePools)
}

func (ts *TestINodePoolSuite) TestMultiNodesEtcd() {
	var clients []*clientv3.Client
	var drivers []driver.DriverV2
	var nodePools []dcron.INodePool

	numberOfNodes := 5
	ServiceName := "TestMultiNodesEtcd"
	updateDuration := 8 * time.Second

	ts.setUpEtcd()
	ts.declareEtcdDrivers(&clients, &drivers, numberOfNodes)

	for i := 0; i < numberOfNodes; i++ {
		nodePools = append(nodePools, dcron.NewNodePool(ServiceName, drivers[i], updateDuration, ts.defaultHashReplicas, nil))
	}
	ts.runCheckJobAvailable(numberOfNodes, ServiceName, &nodePools, updateDuration)
	ts.stopAllNodePools(nodePools)
}

func (ts *TestINodePoolSuite) TestMultiNodesRedisZSet() {
	var clients []*redis.Client
	var drivers []driver.DriverV2
	var nodePools []dcron.INodePool

	numberOfNodes := 5
	ServiceName := "TestMultiNodesZSet"
	updateDuration := 2 * time.Second

	ts.setUpRedis()
	ts.declareRedisZSetDrivers(&clients, &drivers, numberOfNodes)

	for i := 0; i < numberOfNodes; i++ {
		nodePools = append(nodePools, dcron.NewNodePool(ServiceName, drivers[i], updateDuration, ts.defaultHashReplicas, nil))
	}
	ts.runCheckJobAvailable(numberOfNodes, ServiceName, &nodePools, updateDuration)
	ts.stopAllNodePools(nodePools)
}

func (ts *TestINodePoolSuite) TestCheckJobAvailableFailedWithNodePoolRingIsNil() {
	np := &dcron.NodePool{}
	np.SetLogger(dlog.NewLoggerForTest(ts.T()))
	_, err := np.CheckJobAvailable("testjob")
	ts.Equal(dcron.ErrNodePoolIsNil, err)
}

func (ts *TestINodePoolSuite) TestStartFailedWithDriverStartError() {
	expectErr := errors.New("driver start error")
	md := &MockDriver{
		StartFunc: func(context.Context) error {
			return expectErr
		},
	}
	np := dcron.NewNodePool(
		"testServiceName",
		md, 3*time.Second,
		ts.defaultHashReplicas,
		dlog.NewLoggerForTest(ts.T()))
	ts.Equal(expectErr, np.Start(context.Background()))
}

func (ts *TestINodePoolSuite) TestStartFailedWithDriverGetNodesError() {
	expectErr := errors.New("driver get nodes error")
	md := &MockDriver{
		GetNodesFunc: func(ctx context.Context) ([]string, error) {
			return nil, expectErr
		},
	}
	np := dcron.NewNodePool(
		"testServiceName",
		md, 3*time.Second,
		ts.defaultHashReplicas,
		dlog.NewLoggerForTest(ts.T()))
	ts.Equal(expectErr, np.Start(context.Background()))
}

func TestTestINodePoolSuite(t *testing.T) {
	s := new(TestINodePoolSuite)
	suite.Run(t, s)
}
