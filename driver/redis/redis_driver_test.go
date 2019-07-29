package redis

import (
	"fmt"
	"testing"
)

func TestRedisDriver_Scan(t *testing.T) {
	rd, err := NewDriver(&Conf{
		Host: "127.0.0.1",
		Port: 6379,
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
			panic(err)
		}
		fmt.Println(ret)
	}
}
