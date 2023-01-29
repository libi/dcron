package dlog

type Logger interface {
	Printf(string, ...interface{})
}
