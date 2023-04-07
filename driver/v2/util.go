package v2

import "github.com/google/uuid"

// GlobalKeyPrefix is global redis key preifx
const GlobalKeyPrefix = "distributed-cron:"
const DefaultNodesChanLength = 10

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
