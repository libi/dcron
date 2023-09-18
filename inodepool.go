package dcron

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNodePoolIsUpgrading = errors.New("nodePool is upgrading")
)

type INodePool interface {
	Start(ctx context.Context) error
	CheckJobAvailable(jobName string) (bool, error)
	Stop(ctx context.Context) error

	GetNodeID() string
	GetLastNodesUpdateTime() time.Time
}
