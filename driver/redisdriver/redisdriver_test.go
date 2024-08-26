package redisdriver_test

import (
	"context"
	"log"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/dcron-contrib/commons"
	"github.com/dcron-contrib/commons/dlog"
	"github.com/libi/dcron/driver/redisdriver"
	redis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func testFuncNewRedisDriver(addr string) commons.DriverV2 {
	log.Printf("redis=%s", addr)
	redisCli := redis.NewClient(&redis.Options{
		Addr: addr,
	})
	return redisdriver.NewDriver(redisCli)
}

func TestRedisDriver_GetNodes(t *testing.T) {
	rds := miniredis.RunT(t)
	drvs := make([]commons.DriverV2, 0)
	N := 10
	for i := 0; i < N; i++ {
		drv := testFuncNewRedisDriver(rds.Addr())
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

func TestRedisDriver_Stop(t *testing.T) {
	var err error
	var nodes []string
	rds := miniredis.RunT(t)
	drv1 := testFuncNewRedisDriver(rds.Addr())
	drv1.Init(t.Name(),
		commons.NewTimeoutOption(5*time.Second),
		commons.NewLoggerOption(dlog.NewLoggerForTest(t)))

	drv2 := testFuncNewRedisDriver(rds.Addr())
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

	<-time.After(5 * time.Second)
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
