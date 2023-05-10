package dcron_test

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/libi/dcron"
	"github.com/libi/dcron/consistenthash"
	"github.com/libi/dcron/driver"
	"github.com/stretchr/testify/suite"
)

type TestINodePoolSuite struct {
	suite.Suite

	rds                 *miniredis.Miniredis
	defaultHashReplicas int
}

func (ts *TestINodePoolSuite) SetupTest() {
	ts.rds = miniredis.RunT(ts.T())
	ts.defaultHashReplicas = 10
}

func (ts *TestINodePoolSuite) TearDownTest() {
	ts.rds.Close()
}

func (ts *TestINodePoolSuite) TestMultiNodes() {
	var clients []*redis.Client
	var drivers []driver.DriverV2
	var nodePools []dcron.INodePool

	NumberOfNodes := 5
	ServiceName := "TestMultiNodes"
	updateDuration := 2 * time.Second

	for i := 0; i < NumberOfNodes; i++ {
		clients = append(clients, redis.NewClient(&redis.Options{
			Addr: ts.rds.Addr(),
		}))
		drivers = append(drivers, driver.NewRedisDriver(clients[i]))
		nodePools = append(nodePools, dcron.NewNodePool(ServiceName, drivers[i], updateDuration, ts.defaultHashReplicas, nil))
	}

	for i := 0; i < NumberOfNodes; i++ {
		err := nodePools[i].Start(context.Background())
		if err != nil {
			ts.T().Fail()
		}
	}
	<-time.After(updateDuration * 2)
	ring := consistenthash.New(ts.defaultHashReplicas, nil)
	for _, v := range nodePools {
		ring.Add(v.GetNodeID())
	}

	for i := 0; i < 10000; i++ {
		for j := 0; j < NumberOfNodes; j++ {
			ts.Require().Equal(
				nodePools[j].CheckJobAvailable(strconv.Itoa(i)),
				(ring.Get(strconv.Itoa(i)) == nodePools[j].GetNodeID()),
			)
		}
	}
}

func TestTestINodePoolSuite(t *testing.T) {
	s := new(TestINodePoolSuite)
	suite.Run(t, s)
}
