package driver_test

import (
	"context"
	"testing"
	"time"

	"github.com/libi/dcron/dlog"
	"github.com/libi/dcron/driver"
	"github.com/stretchr/testify/require"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/tests/v3/integration"
)

func testFuncNewEtcdDriver(cfg clientv3.Config) driver.DriverV2 {
	cli, err := clientv3.New(cfg)
	if err != nil {
		panic(err)
	}
	return driver.NewEtcdDriver(cli)
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
		drv.Init(t.Name(), driver.NewTimeoutOption(5*time.Second), driver.NewLoggerOption(dlog.NewLoggerForTest(t)))
		err := drv.Start(context.Background())
		require.Nil(t, err)
		drvs = append(drvs, drv)
	}
	<-time.After(5 * time.Second)
	for _, v := range drvs {
		nodes, err := v.GetNodes(context.Background())
		require.Nil(t, err)
		require.Equal(t, N, len(nodes))
	}

	for _, v := range drvs {
		v.Stop(context.Background())
	}
}

func TestEtcdDriver_Stop(t *testing.T) {
	etcdsvr := integration.NewLazyCluster()
	defer etcdsvr.Terminate()

	drv1 := testFuncNewEtcdDriver(clientv3.Config{
		Endpoints:   etcdsvr.EndpointsV3(),
		DialTimeout: 3 * time.Second,
	})
	drv1.Init(t.Name(), driver.NewTimeoutOption(5*time.Second), driver.NewLoggerOption(dlog.NewLoggerForTest(t)))

	drv2 := testFuncNewEtcdDriver(clientv3.Config{
		Endpoints:   etcdsvr.EndpointsV3(),
		DialTimeout: 3 * time.Second,
	})
	drv2.Init(t.Name(), driver.NewTimeoutOption(5*time.Second), driver.NewLoggerOption(dlog.NewLoggerForTest(t)))

	require.Nil(t, drv1.Start(context.Background()))
	require.Nil(t, drv2.Start(context.Background()))

	checkNodesFunc := func(drv driver.DriverV2, count int, timeout time.Duration) {
		tick := time.Tick(3 * time.Second)
		timeoutPoint := time.Now().Add(timeout)
		for range tick {
			if timeoutPoint.Before(time.Now()) {
				t.Fatal("timeout")
			}
			nodes, err := drv.GetNodes(context.Background())
			require.Nil(t, err)
			if len(nodes) == count {
				return
			}
		}
	}

	checkNodesFunc(drv1, 2, 20*time.Second)
	checkNodesFunc(drv2, 2, 20*time.Second)

	drv1.Stop(context.Background())
	checkNodesFunc(drv2, 1, 20*time.Second)

	require.Nil(t, drv1.Start(context.Background()))
	checkNodesFunc(drv2, 2, 20*time.Second)
	checkNodesFunc(drv1, 2, 20*time.Second)

	drv2.Stop(context.Background())
	drv1.Stop(context.Background())
}
