package cron

import (
	"io"
	"log"
	"os"

	"github.com/dcron-contrib/commons/dlog"
)

// DefaultLogger is used by Cron if none is specified.
var DefaultLogger dlog.Logger = dlog.DefaultPrintfLogger(log.New(os.Stdout, "cron: ", log.LstdFlags))

// DiscardLogger can be used by callers to discard all log messages.
var DiscardLogger dlog.Logger = dlog.DefaultPrintfLogger(log.New(io.Discard, "", 0))
