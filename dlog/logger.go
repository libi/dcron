package dlog

import "log"

type PrintfLogger interface {
	Printf(string, ...interface{})
}

type Logger interface {
	PrintfLogger
	Infof(string, ...interface{})
	Warnf(string, ...interface{})
	Errorf(string, ...interface{})
}

type StdLogger struct {
	Log *log.Logger
}

func (l *StdLogger) Infof(format string, args ...interface{}) {
	l.Log.Printf("[INFO] "+format, args...)
}

func (l *StdLogger) Warnf(format string, args ...interface{}) {
	l.Log.Printf("[WARN] "+format, args...)
}

func (l *StdLogger) Errorf(format string, args ...interface{}) {
	l.Log.Printf("[ERROR] "+format, args...)
}

func (l *StdLogger) Printf(format string, args ...interface{}) {
	l.Log.Printf(format, args...)
}
