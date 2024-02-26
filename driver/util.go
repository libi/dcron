package driver

import (
	"time"

	"github.com/google/uuid"
)

// GlobalKeyPrefix is a global redis key prefix
const GlobalKeyPrefix = "distributed-cron:"

func GetKeyPre(serviceName string) string {
	return GlobalKeyPrefix + serviceName + ":"
}

func GetNodeId(serviceName string) string {
	return GetKeyPre(serviceName) + uuid.New().String()
}

func TimePre(t time.Time, preDuration time.Duration) int64 {
	return t.Add(-preDuration).Unix()
}
