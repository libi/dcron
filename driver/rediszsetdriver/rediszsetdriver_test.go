package rediszsetdriver_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/libi/dcron/commons"
	"github.com/libi/dcron/commons/dlog"
	"github.com/libi/dcron/driver/rediszsetdriver"

	redis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func testFuncNewRedisZSetDriver(addr string) commons.DriverV2 {
	redisCli := redis.NewClient(&redis.Options{
		Addr: addr,
	})
	return rediszsetdriver.NewDriver(redisCli)
}

func TestRedisZSetDriver_GetNodes(t *testing.T) {
	rds := miniredis.RunT(t)
	drvs := make([]commons.DriverV2, 0)
	N := 10
	for i := 0; i < N; i++ {
		drv := testFuncNewRedisZSetDriver(rds.Addr())
		drv.Init(
			t.Name(),
			commons.NewTimeoutOption(5*time.Second),
			commons.NewLoggerOption(dlog.NewLoggerForTest(t)))
		err := drv.Start(context.Background())
		require.Nil(t, err)
		drvs = append(drvs, drv)
	}

	for _, v := range drvs {
		nodes, err := v.GetNodes(context.Background())
		require.Nil(t, err)
		require.Equal(t, N, len(nodes))
	}

	for _, v := range drvs {
		v.Stop(context.Background())
	}
}

func TestRedisZSetDriver_Stop(t *testing.T) {
	var err error
	var nodes []string
	rds := miniredis.RunT(t)
	drv1 := testFuncNewRedisZSetDriver(rds.Addr())
	drv1.Init(t.Name(),
		commons.NewTimeoutOption(5*time.Second),
		commons.NewLoggerOption(dlog.NewLoggerForTest(t)))

	drv2 := testFuncNewRedisZSetDriver(rds.Addr())
	drv2.Init(t.Name(),
		commons.NewTimeoutOption(5*time.Second),
		commons.NewLoggerOption(dlog.NewLoggerForTest(t)))

	require.Nil(t, drv2.Start(context.Background()))
	require.Nil(t, drv1.Start(context.Background()))

	nodes, err = drv1.GetNodes(context.Background())
	require.Nil(t, err)
	require.Len(t, nodes, 2)

	nodes, err = drv2.GetNodes(context.Background())
	require.Nil(t, err)
	require.Len(t, nodes, 2)

	drv1.Stop(context.Background())

	<-time.After(6 * time.Second)
	nodes, err = drv2.GetNodes(context.Background())
	require.Nil(t, err)
	require.Len(t, nodes, 1)

	err = drv1.Start(context.Background())
	require.Nil(t, err)
	<-time.After(5 * time.Second)
	nodes, err = drv2.GetNodes(context.Background())
	require.Nil(t, err)
	require.Len(t, nodes, 2)
	nodes, err = drv1.GetNodes(context.Background())
	require.Nil(t, err)
	require.Len(t, nodes, 2)

	drv2.Stop(context.Background())
	drv1.Stop(context.Background())
}
