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
	N := 10
	drvs := make([]driver.DriverV2, N)
	for i := 0; i < N; i++ {
		drv := testFuncNewEtcdDriver(clientv3.Config{
			Endpoints:   etcdsvr.EndpointsV3(),
			DialTimeout: 3 * time.Second,
		})
		drv.Init(t.Name(), driver.NewTimeoutOption(5*time.Second), driver.NewLoggerOption(dlog.NewLoggerForTest(t)))
		err := drv.Start(context.Background())
		require.Nil(t, err)
		drvs[i] = drv
	}
	<-time.After(10 * time.Second)
	for _, drv := range drvs {
		nodes, err := drv.GetNodes(context.Background())
		require.Nil(t, err)
		require.Equal(t, N, len(nodes))
	}

	for _, drv := range drvs {
		drv.Stop(context.Background())
	}
	etcdsvr.Terminate()
}

func TestEtcdDriver_Stop(t *testing.T) {
	var err error
	etcdsvr := integration.NewLazyCluster()

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
	err = drv2.Start(context.Background())
	require.Nil(t, err)

	checkNodesFunc := func(drv driver.DriverV2, count int, timeout time.Duration) {
		tick := time.Tick(5 * time.Second)
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

	err = drv1.Start(context.Background())
	require.Nil(t, err)
	checkTimeout := 20 * time.Second
	checkNodesFunc(drv1, 2, checkTimeout)
	checkNodesFunc(drv2, 2, checkTimeout)

	drv1.Stop(context.Background())
	checkNodesFunc(drv2, 1, checkTimeout)

	err = drv1.Start(context.Background())
	require.Nil(t, err)
	checkNodesFunc(drv1, 2, checkTimeout)
	checkNodesFunc(drv2, 2, checkTimeout)

	drv2.Stop(context.Background())
	drv1.Stop(context.Background())

	etcdsvr.Terminate()
}
