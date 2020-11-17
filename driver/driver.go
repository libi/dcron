package driver

import "time"

//Driver is a driver interface
type Driver interface {
	// Ping is check dirver is valid
	Ping() error
	SetHeartBeat(nodeID string)
	SetTimeout(timeout time.Duration)
	GetServiceNodeList(ServiceName string) ([]string, error)
	RegisterServiceNode(ServiceName string) (string, error)
}
