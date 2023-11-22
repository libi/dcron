package cron

import (
	"io"
	"log"
	"os"

	"github.com/libi/dcron/dlog"
)

// DefaultLogger is used by Cron if none is specified.
var DefaultLogger dlog.Logger = PrintfLogger(log.New(os.Stdout, "cron: ", log.LstdFlags))

// DiscardLogger can be used by callers to discard all log messages.
var DiscardLogger dlog.Logger = PrintfLogger(log.New(io.Discard, "", 0))

// PrintfLogger wraps a Printf-based logger (such as the standard library "log")
// into an implementation of the Logger interface which logs errors only.
func PrintfLogger(l dlog.PrintfLogger) dlog.Logger {
	return &dlog.StdLogger{Log: l}
}

// VerbosePrintfLogger wraps a Printf-based logger (such as the standard library
// "log") into an implementation of the Logger interface which logs everything.
func VerbosePrintfLogger(l dlog.PrintfLogger) dlog.Logger {
	return &dlog.StdLogger{Log: l}
}
