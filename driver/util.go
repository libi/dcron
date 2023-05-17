package driver

import (
	"time"

	"github.com/google/uuid"
)

// GlobalKeyPrefix is global redis key preifx
const GlobalKeyPrefix = "distributed-cron:"

func GetKeyPre(serviceName string) string {
	return GlobalKeyPrefix + serviceName + ":"
}

func GetNodeId(serviceName string) string {
	return GetKeyPre(serviceName) + uuid.New().String()
}

func GetStableJobStore(serviceName string) string {
	return GetKeyPre(serviceName) + "stable-jobs"
}

func GetStableJobStoreTxKey(serviceName string) string {
	return GetKeyPre(serviceName) + "TX:stable-jobs"
}

func TimePre(t time.Time, preDuration time.Duration) int64 {
	return t.Unix() - int64(preDuration.Seconds())
}
