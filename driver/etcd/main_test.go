package etcd

import (
	"flag"
	"os"
	"testing"

	"github.com/golang/glog"
)

func TestMain(m *testing.M) {
	testing.Init()
	_ = flag.Set("v", "12")
	_ = flag.Set("logtostderr", "true")
	flag.Parse()
	defer glog.Flush()

	os.Exit(m.Run())
}
