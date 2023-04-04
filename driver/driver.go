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
	Init(serviceName string, timeout time.Duration, logger dlog.Logger)
	Start() (nodesChan chan []string, err error)
}
