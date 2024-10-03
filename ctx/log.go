package ctx

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/DimmyJing/valise/attr"
	"github.com/DimmyJing/valise/log"
)

const loggerContextKey contextKey = "logger"

func (c Context) WithLog(logger *log.Logger) Context {
	return c.WithValue(loggerContextKey, logger)
}

func (c Context) GetLog() *log.Logger {
	return ValueDefault(c, loggerContextKey, log.Default())
}

func (c Context) Trace(msg any, args ...attr.Attr) {
	c.LogHelper(log.LevelTrace, msg, 0, args...)
}

func (c Context) Tracef(msg string, args ...any) {
	c.LogHelper(log.LevelTrace, fmt.Sprintf(msg, args...), 0)
}

func (c Context) Debug(msg any, args ...attr.Attr) {
	c.LogHelper(log.LevelDebug, msg, 0, args...)
}

func (c Context) Debugf(msg string, args ...any) {
	c.LogHelper(log.LevelDebug, fmt.Sprintf(msg, args...), 0)
}

func (c Context) Info(msg any, args ...attr.Attr) {
	c.LogHelper(log.LevelInfo, msg, 0, args...)
}

func (c Context) Infof(msg string, args ...any) {
	c.LogHelper(log.LevelInfo, fmt.Sprintf(msg, args...), 0)
}

func (c Context) Warn(msg any, args ...attr.Attr) {
	c.LogHelper(log.LevelWarn, msg, 0, args...)
}

func (c Context) Warnf(msg string, args ...any) {
	c.LogHelper(log.LevelWarn, fmt.Sprintf(msg, args...), 0)
}

func (c Context) Error(msg any, args ...attr.Attr) {
	c.LogHelper(log.LevelError, msg, 0, args...)
}

func (c Context) Errorf(msg string, args ...any) {
	c.LogHelper(log.LevelError, fmt.Sprintf(msg, args...), 0)
}

func (c Context) Fatal(msg any, args ...attr.Attr) {
	c.LogHelper(log.LevelFatal, msg, 0, args...)
}

func (c Context) Fatalf(msg string, args ...any) {
	c.LogHelper(log.LevelFatal, fmt.Sprintf(msg, args...), 0)
}

func (c Context) Panic(err error, args ...attr.Attr) {
	c.LogHelper(log.LevelFatal, err, 0, args...)
	panic(err)
}

func (c Context) Panicf(msg string, args ...any) {
	//nolint:goerr113
	err := fmt.Errorf(msg, args...)
	c.LogHelper(log.LevelFatal, err, 0)
	panic(err)
}

func (c Context) Capture(err error, args ...attr.Attr) error {
	c.LogHelper(log.LevelError, err, 0, args...)

	return err
}

func (c Context) Capturef(msg string, args ...any) error {
	//nolint:goerr113
	err := fmt.Errorf(msg, args...)
	c.LogHelper(log.LevelError, err, 0)

	return err
}

type CallerError struct {
	error
	pc uintptr
}

func trimCallerPath(path string, numTrim int) string {
	if numTrim <= 0 {
		return path
	}

	idx := strings.LastIndexByte(path, '/')
	if idx == -1 {
		return path
	}

	for range numTrim - 1 {
		idx = strings.LastIndexByte(path[:idx], '/')
		if idx == -1 {
			return path
		}
	}

	return path[idx+1:]
}

func (e *CallerError) Error() string {
	fs := runtime.CallersFrames([]uintptr{e.pc})
	f, _ := fs.Next()
	//nolint:mnd
	return fmt.Sprintf("<%s:%d>{%s}", trimCallerPath(f.File, 2), f.Line, e.error.Error())
}

func (e *CallerError) Unwrap() error {
	return e.error
}

func newWithStack(err error) error {
	var pcs [1]uintptr

	//nolint:mnd
	runtime.Callers(3, pcs[:])

	return &CallerError{
		error: err,
		pc:    pcs[0],
	}
}

func (c Context) Fail(err error, args ...attr.Attr) error {
	err = newWithStack(err)
	c.recordEvent(err, args)
	c.fail(err.Error())

	return err
}

func (c Context) Failf(msg string, args ...any) error {
	//nolint:goerr113
	err := newWithStack(fmt.Errorf(msg, args...))
	c.recordEvent(err, []attr.Attr{})
	c.fail(err.Error())

	return err
}

func (c Context) FailIf(err error, args ...attr.Attr) error {
	if err != nil {
		err = newWithStack(err)
		c.recordEvent(err, args)
		c.fail(err.Error())

		return err
	}

	return nil
}

func (c Context) LogHelper(
	level log.Level,
	msg any,
	skips int,
	args ...attr.Attr,
) {
	if c.GetLog().Enabled(c, level) {
		c.recordEvent(msg, args)
		c.GetLog().LogHelper(c, level, msg, skips+1, args...)
	}
}
