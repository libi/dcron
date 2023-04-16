package v2_test

import (
	"testing"
	"time"

	"github.com/libi/dcron/driver"
	v2 "github.com/libi/dcron/driver/v2"
	"github.com/stretchr/testify/require"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/tests/v3/integration"
)

func testFuncNewEtcdDriver(cfg clientv3.Config) driver.DriverV2 {
	cli, err := clientv3.New(cfg)
	if err != nil {
		panic(err)
	}
	return v2.NewEtcdDriver(cli)
}

func TestEtcdDriver_GetNodes(t *testing.T) {
	etcdsvr := integration.NewLazyCluster()
	defer etcdsvr.Terminate()
	N := 10
	drvs := make([]driver.DriverV2, 0)
	for i := 0; i < N; i++ {
		drv := testFuncNewEtcdDriver(clientv3.Config{
			Endpoints:   etcdsvr.EndpointsV3(),
			DialTimeout: 3 * time.Second,
		})
		drv.Init(t.Name(), 5*time.Second, nil)
		err := drv.Start()
		require.Nil(t, err)
		drvs = append(drvs, drv)
	}
	<-time.After(5 * time.Second)
	for _, v := range drvs {
		nodes, err := v.GetNodes()
		require.Nil(t, err)
		require.Equal(t, N, len(nodes))
	}

	for _, v := range drvs {
		v.Stop()
	}
}
