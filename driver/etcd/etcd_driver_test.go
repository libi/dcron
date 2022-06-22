package etcd

import (
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestEtcdDriver(t *testing.T) {

	var endpoints = []string{"127.0.0.1:2379"}

	ed, err := NewEtcdDriver(endpoints)

	require.Nil(t, err)
	serviceName := "testService"

	nodeID, err := ed.RegisterServiceNode(serviceName)

	require.Nil(t, err)

	t.Logf("nodeId:%v", nodeID)

	time.Sleep(time.Second * 1)

	//register second node
	ed.RegisterServiceNode(serviceName)

	list, err := ed.GetServiceNodeList(serviceName)

	require.Nil(t, err)

	for _, v := range list {
		t.Logf("item:%v", v)
	}

	time.Sleep(time.Second * 10)

}
