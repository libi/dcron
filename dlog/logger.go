package dlog

import (
	"testing"
)

type PrintfLogger interface {
	Printf(string, ...interface{})
}

type LogfLogger interface {
	Logf(string, ...interface{})
}

type Logger interface {
	PrintfLogger
	Infof(string, ...interface{})
	Warnf(string, ...interface{})
	Errorf(string, ...interface{})
}

type StdLogger struct {
	Log PrintfLogger
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

type PrintfLoggerFromLogfLogger struct {
	Log LogfLogger
}

func (l *PrintfLoggerFromLogfLogger) Printf(fmt string, args ...interface{}) {
	l.Log.Logf(fmt, args)
}

func NewPrintfLoggerFromLogfLogger(logger LogfLogger) PrintfLogger {
	return &PrintfLoggerFromLogfLogger{Log: logger}
}

func NewLoggerForTest(t *testing.T) Logger {
	return &StdLogger{
		Log: NewPrintfLoggerFromLogfLogger(t),
	}
}
