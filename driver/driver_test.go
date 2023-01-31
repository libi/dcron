package driver_test

import (
	"flag"
	"testing"

	"github.com/go-redis/redis/v8"
	"github.com/libi/dcron/driver"
	RedisDriver "github.com/libi/dcron/driver/redis"
	"github.com/stretchr/testify/require"
)

var (
	redisAddr = flag.String("rAddr", "127.0.0.1:6379", "redis serve addr")
)

// you should run this test when the redis is served.
// you can run test like below command.
// go test -v --rAddr 127.0.0.1:6379
// rAddr is the redis serve addr

func Ping(d driver.Driver) error {
	return d.Ping()
}

func TestRedisDriver(t *testing.T) {
	t.Logf("test redis serve on %s", *redisAddr)
	driver, err := RedisDriver.NewDriver(&redis.Options{
		Addr: *redisAddr,
	})
	require.Nil(t, err)
	require.Nil(t, Ping(driver))
}
