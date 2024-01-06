package dlog

import "go.uber.org/zap"

type ZapLoggerImpl struct {
	log *zap.Logger
}

func (l *ZapLoggerImpl) Printf(format string, args ...any) {
	l.log.Sugar().Infof(format, args...)
}

func (l *ZapLoggerImpl) Infof(format string, args ...any) {
	l.log.Sugar().Infof(format, args...)
}

func (l *ZapLoggerImpl) Warnf(format string, args ...any) {
	l.log.Sugar().Warnf(format, args...)
}

func (l *ZapLoggerImpl) Errorf(format string, args ...any) {
	l.log.Sugar().Errorf(format, args...)
}

func NewZapLogger(logger *zap.Logger) Logger {
	return &ZapLoggerImpl{log: logger}
}
