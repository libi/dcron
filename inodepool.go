package dcron

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNodePoolIsUpgrading = errors.New("nodePool is upgrading")
	ErrNodePoolIsNil       = errors.New("nodePool is nil")
)

type INodePool interface {
	Start(ctx context.Context) error
	CheckJobAvailable(jobName string) (bool, error)
	Stop(ctx context.Context) error

	GetNodeID() string
	GetLastNodesUpdateTime() time.Time
}
