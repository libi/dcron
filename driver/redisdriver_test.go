package driver_test

import (
	"log"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/libi/dcron/dlog"
	"github.com/libi/dcron/driver"
	"github.com/stretchr/testify/require"
)

func testFuncNewRedisDriver(addr string) driver.DriverV2 {
	log.Println("redis=", addr)
	redisCli := redis.NewClient(&redis.Options{
		Addr: addr,
	})
	return driver.NewRedisDriver(redisCli)
}

func TestRedisDriver_GetNodes(t *testing.T) {
	rds := miniredis.RunT(t)
	drvs := make([]driver.DriverV2, 0)
	N := 10
	for i := 0; i < N; i++ {
		drv := testFuncNewRedisDriver(rds.Addr())
		drv.Init(
			t.Name(),
			driver.NewTimeoutOption(5*time.Second),
			driver.NewLoggerOption(dlog.NewLoggerForTest(t)))
		err := drv.Start()
		require.Nil(t, err)
		drvs = append(drvs, drv)
	}

	for _, v := range drvs {
		nodes, err := v.GetNodes()
		require.Nil(t, err)
		require.Equal(t, N, len(nodes))
	}

	for _, v := range drvs {
		v.Stop()
	}
}

func TestRedisDriver_Stop(t *testing.T) {
	var err error
	var nodes []string
	rds := miniredis.RunT(t)
	drv1 := testFuncNewRedisDriver(rds.Addr())
	drv1.Init(t.Name(),
		driver.NewTimeoutOption(5*time.Second),
		driver.NewLoggerOption(dlog.NewLoggerForTest(t)))

	drv2 := testFuncNewRedisDriver(rds.Addr())
	drv2.Init(t.Name(),
		driver.NewTimeoutOption(5*time.Second),
		driver.NewLoggerOption(dlog.NewLoggerForTest(t)))
	err = drv2.Start()
	require.Nil(t, err)

	err = drv1.Start()
	require.Nil(t, err)

	nodes, err = drv1.GetNodes()
	require.Nil(t, err)
	require.Len(t, nodes, 2)

	nodes, err = drv2.GetNodes()
	require.Nil(t, err)
	require.Len(t, nodes, 2)

	drv1.Stop()

	<-time.After(5 * time.Second)
	nodes, err = drv2.GetNodes()
	require.Nil(t, err)
	require.Len(t, nodes, 1)

	err = drv1.Start()
	require.Nil(t, err)
	<-time.After(5 * time.Second)
	nodes, err = drv2.GetNodes()
	require.Nil(t, err)
	require.Len(t, nodes, 2)

	drv2.Stop()
}
