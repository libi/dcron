package commons

import (
	"time"

	"github.com/libi/dcron/commons/dlog"
)

const (
	OptionTypeTimeout = 0x600
	OptionTypeLogger  = 0x601
)

type Option interface {
	Type() int
}

type TimeoutOption struct{ Timeout time.Duration }

func (to TimeoutOption) Type() int                         { return OptionTypeTimeout }
func NewTimeoutOption(timeout time.Duration) TimeoutOption { return TimeoutOption{Timeout: timeout} }

type LoggerOption struct{ Logger dlog.Logger }

func (to LoggerOption) Type() int                     { return OptionTypeLogger }
func NewLoggerOption(logger dlog.Logger) LoggerOption { return LoggerOption{Logger: logger} }
