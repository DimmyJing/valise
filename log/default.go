package log

import (
	"context"
	"fmt"

	"github.com/DimmyJing/valise/attr"
)

//nolint:gochecknoglobals
var logger = New()

func Default() *Logger {
	return logger
}

func SetDefault(l *Logger) {
	logger = l
}

func With(args ...attr.Attr) *Logger {
	return logger.With(args...)
}

func Trace(msg any, args ...attr.Attr) {
	logger.LogHelper(context.Background(), LevelTrace, msg, 0, args...)
}

func Tracef(msg string, args ...any) {
	logger.LogHelper(context.Background(), LevelTrace, fmt.Sprintf(msg, args...), 0)
}

func TraceContext(ctx context.Context, msg any, args ...attr.Attr) {
	logger.LogHelper(ctx, LevelTrace, msg, 0, args...)
}

func TracefContext(ctx context.Context, msg string, args ...any) {
	logger.LogHelper(ctx, LevelTrace, fmt.Sprintf(msg, args...), 0)
}

func Debug(msg any, args ...attr.Attr) {
	logger.LogHelper(context.Background(), LevelDebug, msg, 0, args...)
}

func Debugf(msg string, args ...any) {
	logger.LogHelper(context.Background(), LevelDebug, fmt.Sprintf(msg, args...), 0)
}

func DebugContext(ctx context.Context, msg any, args ...attr.Attr) {
	logger.LogHelper(ctx, LevelDebug, msg, 0, args...)
}

func DebugfContext(ctx context.Context, msg string, args ...any) {
	logger.LogHelper(ctx, LevelDebug, fmt.Sprintf(msg, args...), 0)
}

func Info(msg any, args ...attr.Attr) {
	logger.LogHelper(context.Background(), LevelInfo, msg, 0, args...)
}

func Infof(msg string, args ...any) {
	logger.LogHelper(context.Background(), LevelInfo, fmt.Sprintf(msg, args...), 0)
}

func InfoContext(ctx context.Context, msg any, args ...attr.Attr) {
	logger.LogHelper(ctx, LevelInfo, msg, 0, args...)
}

func InfofContext(ctx context.Context, msg string, args ...any) {
	logger.LogHelper(ctx, LevelInfo, fmt.Sprintf(msg, args...), 0)
}

func Warn(msg any, args ...attr.Attr) {
	logger.LogHelper(context.Background(), LevelWarn, msg, 0, args...)
}

func Warnf(msg string, args ...any) {
	logger.LogHelper(context.Background(), LevelWarn, fmt.Sprintf(msg, args...), 0)
}

func WarnContext(ctx context.Context, msg any, args ...attr.Attr) {
	logger.LogHelper(ctx, LevelWarn, msg, 0, args...)
}

func WarnfContext(ctx context.Context, msg string, args ...any) {
	logger.LogHelper(ctx, LevelWarn, fmt.Sprintf(msg, args...), 0)
}

func Error(msg any, args ...attr.Attr) {
	logger.LogHelper(context.Background(), LevelError, msg, 0, args...)
}

func Errorf(msg string, args ...any) {
	logger.LogHelper(context.Background(), LevelError, fmt.Sprintf(msg, args...), 0)
}

func ErrorContext(ctx context.Context, msg any, args ...attr.Attr) {
	logger.LogHelper(ctx, LevelError, msg, 0, args...)
}

func ErrorfContext(ctx context.Context, msg string, args ...any) {
	logger.LogHelper(ctx, LevelError, fmt.Sprintf(msg, args...), 0)
}

func Fatal(msg any, args ...attr.Attr) {
	logger.LogHelper(context.Background(), LevelFatal, msg, 0, args...)
}

func Fatalf(msg string, args ...any) {
	logger.LogHelper(context.Background(), LevelFatal, fmt.Sprintf(msg, args...), 0)
}

func FatalContext(ctx context.Context, msg any, args ...attr.Attr) {
	logger.LogHelper(ctx, LevelFatal, msg, 0, args...)
}

func FatalfContext(ctx context.Context, msg string, args ...any) {
	logger.LogHelper(ctx, LevelFatal, fmt.Sprintf(msg, args...), 0)
}

func Panic(err error, args ...attr.Attr) {
	logger.LogHelper(context.Background(), LevelFatal, err, 0, args...)
	panic(err)
}

func Panicf(msg string, args ...any) {
	//nolint:goerr113
	err := fmt.Errorf(msg, args...)
	logger.LogHelper(context.Background(), LevelFatal, err, 0)
	panic(err)
}

func PanicContext(ctx context.Context, err error, args ...attr.Attr) {
	logger.LogHelper(ctx, LevelFatal, err, 0, args...)
	panic(err)
}

func PanicfContext(ctx context.Context, msg string, args ...any) {
	//nolint:goerr113
	err := fmt.Errorf(msg, args...)
	logger.LogHelper(ctx, LevelFatal, err, 0)
	panic(err)
}
