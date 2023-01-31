package redis

import (
	"testing"

	"github.com/go-redis/redis/v8"
)

func TestRedisDriver_Scan(t *testing.T) {
	rd, err := NewDriver(&redis.Options{
		Addr: "127.0.0.1:6379",
	})
	if err != nil {
		return
	}
	testStr := []string{
		"*", "-----", "", "!@#$%^", "1", "false",
	}
	for _, str := range testStr {
		ret, err := rd.scan(str)
		if err != nil {
			t.Error(err)
		}
		t.Log(ret)
	}
}
