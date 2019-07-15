package driver

import "time"

//DriverConnOpt is driver's option
type DriverConnOpt struct {
	Host     string
	Port     string
	Password string
}

//Driver is a driver interface
type Driver interface {
	Open(dataSourceOption DriverConnOpt)
	SetHeartBeat(nodeID string)
	SetTimeout(timeout time.Duration)
	GetServiceNodeList(ServiceName string) ([]string, error)
	RegisterServiceNode(ServiceName string) string
}

var (
	drivers = map[string]Driver{}
)

//RegisterDriver register a driver
func RegisterDriver(driverName string, driver Driver) {
	if driver == nil {
		panic("driver is nil")
	}
	if _, hava := drivers[driverName]; hava {
		panic("driver exists" + driverName)
	}
	drivers[driverName] = driver
}

//GetDriver get a driver
func GetDriver(driverName string) Driver {
	return drivers[driverName]
}
