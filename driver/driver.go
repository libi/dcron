package driver

import (
	"time"

	"github.com/libi/dcron/dlog"
)

//Driver is a driver interface
type Driver interface {
	// Ping is check dirver is valid
	Ping() error
	SetLogger(log dlog.Logger)
	SetHeartBeat(nodeID string)
	SetTimeout(timeout time.Duration)
	GetServiceNodeList(ServiceName string) ([]string, error)
	RegisterServiceNode(ServiceName string) (string, error)
}

type DriverV2 interface {
	// init driver
	Init(serviceName string, timeout time.Duration, logger dlog.Logger)
	// get nodeID
	NodeID() string
	// get nodes
	GetNodes() (nodes []string, err error)
	Start() (err error)
	Stop() (err error)
}
