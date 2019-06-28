package driver

import "time"

type DriverConnOpt struct {
	Host string
	Port string
	Password string
}
type Driver interface {
	Open(dataSourceOption DriverConnOpt)
	SetHeartBeat(nodeId string)
	SetTimeout(timeout time.Duration)
	GetServiceNodeList(ServiceName string)([]string,error)
	RegisterServiceNode(ServiceName string)(string)
}

var (
	drivers = map[string]Driver{}
)

func RegisterDriver(driverName string, driver Driver) {
	if driver == nil {
		panic("driver is nil")
	}
	if _, hava := drivers[driverName]; hava {
		panic("driver exists" + driverName)
	}
	drivers[driverName] = driver
}

func GetDriver(driverName string) Driver {
	return drivers[driverName]
}

