package driver

import "context"

// This is a mock driver used for unit test.

type MockDriver struct {
	StartFunc    func(context.Context) error
	GetNodesFunc func(context.Context) ([]string, error)
}

func (md *MockDriver) Init(serviceName string, opts ...Option) {}

func (md *MockDriver) NodeID() string {
	return ""
}

func (md *MockDriver) GetNodes(ctx context.Context) (nodes []string, err error) {
	if md.GetNodesFunc != nil {
		return md.GetNodesFunc(ctx)
	}
	return
}

func (md *MockDriver) Start(ctx context.Context) (err error) {
	if md.StartFunc != nil {
		return md.StartFunc(ctx)
	}
	return
}

func (md *MockDriver) Stop(ctx context.Context) (err error) {
	return
}

func (md *MockDriver) withOption(opt Option) (err error) {
	return
}
