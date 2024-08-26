package dcron_test

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/libi/dcron"
	"github.com/libi/dcron/commons/dlog"
	"github.com/libi/dcron/consistenthash"
	"github.com/stretchr/testify/suite"
)

type testINodePoolSuite struct {
	suite.Suite

	defaultHashReplicas int
}

func (ts *testINodePoolSuite) SetupTest() {
	ts.defaultHashReplicas = 10
}

func (ts *testINodePoolSuite) TearDownTest() {

}

func (ts *testINodePoolSuite) stopAllNodePools(nodePools []dcron.INodePool) {
	for _, nodePool := range nodePools {
		nodePool.Stop(context.Background())
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

func (ts *testINodePoolSuite) TestCheckJobAvailableFailedWithNodePoolRingIsNil() {
	np := &dcron.NodePool{}
	np.SetLogger(dlog.NewLoggerForTest(ts.T()))
	_, err := np.CheckJobAvailable("testjob")
	ts.Equal(dcron.ErrNodePoolIsNil, err)
}

func (ts *testINodePoolSuite) TestStartFailedWithDriverStartError() {
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

func (ts *testINodePoolSuite) TestStartFailedWithDriverGetNodesError() {
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
	s := new(testINodePoolSuite)
	suite.Run(t, s)
}
