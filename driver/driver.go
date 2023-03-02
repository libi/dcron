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

	SupportStableJob() bool
	StableJobDriver
}

type StableJobDriver interface {
	Store(serviceName string, key string, body []byte) (err error)
	Get(serviceName string, key string) (body []byte, err error)
	Remove(serviceName string, key string) (err error)
}
