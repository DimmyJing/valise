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
	logger.LogHelper(context.Background(), LevelTrace, msg, args...)
}

func Tracef(msg string, args ...any) {
	logger.LogHelper(context.Background(), LevelTrace, fmt.Sprintf(msg, args...))
}

func TraceContext(ctx context.Context, msg any, args ...attr.Attr) {
	logger.LogHelper(ctx, LevelTrace, msg, args...)
}

func TracefContext(ctx context.Context, msg string, args ...any) {
	logger.LogHelper(ctx, LevelTrace, fmt.Sprintf(msg, args...))
}

func Debug(msg any, args ...attr.Attr) {
	logger.LogHelper(context.Background(), LevelDebug, msg, args...)
}

func Debugf(msg string, args ...any) {
	logger.LogHelper(context.Background(), LevelDebug, fmt.Sprintf(msg, args...))
}

func DebugContext(ctx context.Context, msg any, args ...attr.Attr) {
	logger.LogHelper(ctx, LevelDebug, msg, args...)
}

func DebugfContext(ctx context.Context, msg string, args ...any) {
	logger.LogHelper(ctx, LevelDebug, fmt.Sprintf(msg, args...))
}

func Info(msg any, args ...attr.Attr) {
	logger.LogHelper(context.Background(), LevelInfo, msg, args...)
}

func Infof(msg string, args ...any) {
	logger.LogHelper(context.Background(), LevelInfo, fmt.Sprintf(msg, args...))
}

func InfoContext(ctx context.Context, msg any, args ...attr.Attr) {
	logger.LogHelper(ctx, LevelInfo, msg, args...)
}

func InfofContext(ctx context.Context, msg string, args ...any) {
	logger.LogHelper(ctx, LevelInfo, fmt.Sprintf(msg, args...))
}

func Warn(msg any, args ...attr.Attr) {
	logger.LogHelper(context.Background(), LevelWarn, msg, args...)
}

func Warnf(msg string, args ...any) {
	logger.LogHelper(context.Background(), LevelWarn, fmt.Sprintf(msg, args...))
}

func WarnContext(ctx context.Context, msg any, args ...attr.Attr) {
	logger.LogHelper(ctx, LevelWarn, msg, args...)
}

func WarnfContext(ctx context.Context, msg string, args ...any) {
	logger.LogHelper(ctx, LevelWarn, fmt.Sprintf(msg, args...))
}

func Error(msg any, args ...attr.Attr) {
	logger.LogHelper(context.Background(), LevelError, msg, args...)
}

func Errorf(msg string, args ...any) {
	logger.LogHelper(context.Background(), LevelError, fmt.Sprintf(msg, args...))
}

func ErrorContext(ctx context.Context, msg any, args ...attr.Attr) {
	logger.LogHelper(ctx, LevelError, msg, args...)
}

func ErrorfContext(ctx context.Context, msg string, args ...any) {
	logger.LogHelper(ctx, LevelError, fmt.Sprintf(msg, args...))
}

func Fatal(msg any, args ...attr.Attr) {
	logger.LogHelper(context.Background(), LevelFatal, msg, args...)
}

func Fatalf(msg string, args ...any) {
	logger.LogHelper(context.Background(), LevelFatal, fmt.Sprintf(msg, args...))
}

func FatalContext(ctx context.Context, msg any, args ...attr.Attr) {
	logger.LogHelper(ctx, LevelFatal, msg, args...)
}

func FatalfContext(ctx context.Context, msg string, args ...any) {
	logger.LogHelper(ctx, LevelFatal, fmt.Sprintf(msg, args...))
}

func Panic(err error, args ...attr.Attr) {
	logger.LogHelper(context.Background(), LevelFatal, err, args...)
	panic(err)
}

func Panicf(msg string, args ...any) {
	//nolint:goerr113
	err := fmt.Errorf(msg, args...)
	logger.LogHelper(context.Background(), LevelFatal, err)
	panic(err)
}

func PanicContext(ctx context.Context, err error, args ...attr.Attr) {
	logger.LogHelper(ctx, LevelFatal, err, args...)
	panic(err)
}

func PanicfContext(ctx context.Context, msg string, args ...any) {
	//nolint:goerr113
	err := fmt.Errorf(msg, args...)
	logger.LogHelper(ctx, LevelFatal, err)
	panic(err)
}
