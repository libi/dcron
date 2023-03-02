package redis_cluster

import (
	"testing"

	"github.com/libi/dcron/driver"
)

func TestClusterScan(t *testing.T) {
	rd, err := NewDriver(&Conf{
		Addrs: []string{"127.0.0.1:6379"},
	})
	if err != nil {
		return
	}
	matchStr := driver.GetKeyPre("service")
	ret, err := rd.scan(matchStr)
	if err != nil {
		t.Log(err)
	}
	t.Log(ret)
}
