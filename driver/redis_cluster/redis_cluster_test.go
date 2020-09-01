package redis_cluster

import "testing"

func TestClusterScan(t *testing.T) {
	rd, err := NewDriver(&Conf{
		Addrs: []string{"172.28.0.3:6379", "172.28.0.4:6379", "172.28.0.5:6379"},
	})
	if err != nil {
		t.Error(err)
	}
	matchStr := rd.getKeyPre("service")
	ret, err := rd.scan(matchStr)
	if err != nil {
		t.Error(err)
	}
	t.Log(ret)
}
