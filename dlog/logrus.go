package dlog

import "github.com/sirupsen/logrus"

type LogrusLoggerImpl struct {
	log *logrus.Logger
}

func (l *LogrusLoggerImpl) Printf(format string, args ...any) {
	l.log.Printf(format, args...)
}

func (l *LogrusLoggerImpl) Infof(format string, args ...any) {
	l.log.Infof(format, args...)
}

func (l *LogrusLoggerImpl) Warnf(format string, args ...any) {
	l.log.Warnf(format, args...)
}

func (l *LogrusLoggerImpl) Errorf(format string, args ...any) {
	l.log.Errorf(format, args...)
}

func NewLogrusLogger(logger *logrus.Logger) Logger {
	return &LogrusLoggerImpl{log: logger}
}
