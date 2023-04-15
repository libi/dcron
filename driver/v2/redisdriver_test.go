package v2_test

import (
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/libi/dcron/driver"
	"github.com/stretchr/testify/require"
)

func TestRedisDriver_GetNodes(t *testing.T) {
	rds := miniredis.RunT(t)
	drvs := make([]driver.DriverV2, 0)
	N := 10
	for i := 0; i < N; i++ {
		redisCli := redis.NewClient(&redis.Options{
			Addr: rds.Addr(),
		})
		drv := driver.NewRedisDriver(redisCli)
		drv.Init(t.Name(), 5*time.Second, nil)
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
