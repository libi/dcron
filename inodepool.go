package dcron

import "context"

type INodePool interface {
	Start(ctx context.Context) error
	CheckJobAvailable(jobName string) bool
	Stop(ctx context.Context) error

	GetNodeID() string
}
