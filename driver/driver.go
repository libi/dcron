package driver

import "time"

//Driver is a driver interface
type Driver interface {
	// Ping is check dirver is valid
	Ping() error
	DoHeartBeat(nodeID string,timeout time.Duration)
	GetServiceNodeList(ServiceName string) ([]string, error)
	RegisterServiceNode(ServiceName string,lifeTime time.Duration) string
	// IsCheckAlive 驱动是否断线，自动才重新注册等情况下存活
	IsCheckAlive() bool
}
