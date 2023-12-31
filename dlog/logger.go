package dlog

import (
	"testing"
)

type PrintfLogger interface {
	Printf(string, ...any)
}

type LogfLogger interface {
	Logf(string, ...any)
}

type Logger interface {
	PrintfLogger
	Infof(string, ...any)
	Warnf(string, ...any)
	Errorf(string, ...any)
}

type StdLogger struct {
	Log        PrintfLogger
	LogVerbose bool
}

func (l *StdLogger) Infof(format string, args ...any) {
	if !l.LogVerbose {
		return
	}
	l.Log.Printf("[INFO] "+format, args...)
}

func (l *StdLogger) Warnf(format string, args ...any) {
	l.Log.Printf("[WARN] "+format, args...)
}

func (l *StdLogger) Errorf(format string, args ...any) {
	l.Log.Printf("[ERROR] "+format, args...)
}

func (l *StdLogger) Printf(format string, args ...any) {
	l.Log.Printf(format, args...)
}

type PrintfLoggerFromLogfLogger struct {
	Log LogfLogger
}

func (l *PrintfLoggerFromLogfLogger) Printf(fmt string, args ...any) {
	l.Log.Logf(fmt, args)
}

func NewPrintfLoggerFromLogfLogger(logger LogfLogger) PrintfLogger {
	return &PrintfLoggerFromLogfLogger{Log: logger}
}

func NewLoggerForTest(t *testing.T) Logger {
	return &StdLogger{
		Log:        NewPrintfLoggerFromLogfLogger(t),
		LogVerbose: true,
	}
}

// 这个方法会打印出所有的WARN level以上的LOG
func WarnPrintfLogger(l PrintfLogger) Logger {
	return &StdLogger{Log: l, LogVerbose: false}
}

// 这个方法会打印出所有的INFO level的LOG
func VerbosePrintfLogger(l PrintfLogger) Logger {
	return &StdLogger{Log: l, LogVerbose: true}
}

// 默认的Logger构造函数，会打印出所有WARN level以上的LOG
func DefaultPrintfLogger(l PrintfLogger) Logger {
	return WarnPrintfLogger(l)
}
