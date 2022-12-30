package etcd

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/tests/v3/integration"
)

func TestEtcdDriver(t *testing.T) {
	var lazyCluster = integration.NewLazyCluster()
	defer lazyCluster.Terminate()

	ed, err := NewEtcdDriver(&clientv3.Config{
		Endpoints:   lazyCluster.EndpointsV3(),
		DialTimeout: dialTimeout,
	})

	require.Nil(t, err)
	serviceName := "testService"

	nodeMap := make(map[string]string)

	count := 10

	for i := 0; i < count; i++ {
		nodeID, err := ed.RegisterServiceNode(serviceName)
		require.Nil(t, err)
		t.Logf("nodeId %v:%v", i, nodeID)
		nodeMap[nodeID] = nodeID
	}

	list, err := ed.GetServiceNodeList(serviceName)

	require.Nil(t, err)

	require.Equal(t, count, len(list))

	for _, v := range list {
		if _, ok := nodeMap[v]; !ok {
			t.Errorf("nodeId %v not found!!!", v)
		}
	}

}

func TestSetHeartBeat(t *testing.T) {

	var lazyCluster = integration.NewLazyCluster()
	defer lazyCluster.Terminate()

	ed, err := NewEtcdDriver(&clientv3.Config{
		Endpoints:   lazyCluster.EndpointsV3(),
		DialTimeout: dialTimeout,
	})

	require.Nil(t, err)
	serviceName := "testService"

	nodeMap := make(map[string]string)

	count := 10

	//一半设置心跳
	for i := 0; i < count; i++ {
		nodeID, err := ed.RegisterServiceNode(serviceName)
		require.Nil(t, err)
		t.Logf("nodeId %v:%v", i, nodeID)
		if i%2 == 0 {
			ed.SetHeartBeat(nodeID)
			nodeMap[nodeID] = nodeID
		}
	}

	time.Sleep(time.Second * 10)

	//10s后获取serverList，预期只能取到一半
	list, err := ed.GetServiceNodeList(serviceName)

	require.Nil(t, err)

	require.Equal(t, len(nodeMap), len(list))

	for _, v := range list {
		if _, ok := nodeMap[v]; !ok {
			t.Errorf("nodeId %v not found!!!", v)
		}
	}

}
