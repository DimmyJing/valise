package log

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"time"

	"github.com/DimmyJing/valise/attr"
	"github.com/DimmyJing/valise/utils"
	"github.com/sanity-io/litter"
)

//nolint:exhaustruct,gochecknoglobals
var lit = litter.Options{
	HidePrivateFields: false,
}

func SetLitterOptions(options litter.Options) {
	lit = options
}

type Logger struct {
	logger *slog.Logger
}

//nolint:gochecknoglobals
var logger = New()

func Default() *Logger {
	return logger
}

func SetDefault(l *Logger) {
	logger = l
}

func New(options ...Option) *Logger {
	return &Logger{slog.New(NewHandler(options...))}
}

func With(args ...attr.Attr) *Logger {
	return logger.With(args...)
}

func (l *Logger) Enabled(ctx context.Context, level Level) bool {
	return l.logger.Enabled(ctx, slog.Level(level))
}

func (l *Logger) Handler() slog.Handler { //nolint:ireturn
	return l.logger.Handler()
}

func (l *Logger) With(args ...attr.Attr) *Logger {
	return &Logger{logger: slog.New(l.logger.Handler().WithAttrs(args))}
}

func (l *Logger) WithGroup(group string) *Logger {
	return &Logger{logger: l.logger.WithGroup(group)}
}

func (l *Logger) Trace(msg any, args ...attr.Attr) {
	l.log(context.Background(), LevelTrace, msg, 0, args)
}

func (l *Logger) Tracef(msg string, args ...any) {
	utils.EmulatePrintf(msg, args...)
	l.log(context.Background(), LevelTrace, fmt.Sprintf(msg, args...), 0, nil)
}

func (l *Logger) TraceContext(ctx context.Context, msg any, args ...attr.Attr) {
	l.log(ctx, LevelTrace, msg, 0, args)
}

func (l *Logger) TracefContext(ctx context.Context, msg string, args ...any) {
	utils.EmulatePrintf(msg, args...)
	l.log(ctx, LevelTrace, fmt.Sprintf(msg, args...), 0, nil)
}

func (l *Logger) Debug(msg any, args ...attr.Attr) {
	l.log(context.Background(), LevelDebug, msg, 0, args)
}

func (l *Logger) Debugf(msg string, args ...any) {
	utils.EmulatePrintf(msg, args...)
	l.log(context.Background(), LevelDebug, fmt.Sprintf(msg, args...), 0, nil)
}

func (l *Logger) DebugContext(ctx context.Context, msg any, args ...attr.Attr) {
	l.log(ctx, LevelDebug, msg, 0, args)
}

func (l *Logger) DebugfContext(ctx context.Context, msg string, args ...any) {
	utils.EmulatePrintf(msg, args...)
	l.log(ctx, LevelDebug, fmt.Sprintf(msg, args...), 0, nil)
}

func (l *Logger) Info(msg any, args ...attr.Attr) {
	l.log(context.Background(), LevelInfo, msg, 0, args)
}

func (l *Logger) Infof(msg string, args ...any) {
	utils.EmulatePrintf(msg, args...)
	l.log(context.Background(), LevelInfo, fmt.Sprintf(msg, args...), 0, nil)
}

func (l *Logger) InfoContext(ctx context.Context, msg any, args ...attr.Attr) {
	l.log(ctx, LevelInfo, msg, 0, args)
}

func (l *Logger) InfofContext(ctx context.Context, msg string, args ...any) {
	utils.EmulatePrintf(msg, args...)
	l.log(ctx, LevelInfo, fmt.Sprintf(msg, args...), 0, nil)
}

func (l *Logger) Warn(msg any, args ...attr.Attr) {
	l.log(context.Background(), LevelWarn, msg, 0, args)
}

func (l *Logger) Warnf(msg string, args ...any) {
	utils.EmulatePrintf(msg, args...)
	l.log(context.Background(), LevelWarn, fmt.Sprintf(msg, args...), 0, nil)
}

func (l *Logger) WarnContext(ctx context.Context, msg any, args ...attr.Attr) {
	l.log(ctx, LevelWarn, msg, 0, args)
}

func (l *Logger) WarnfContext(ctx context.Context, msg string, args ...any) {
	utils.EmulatePrintf(msg, args...)
	l.log(ctx, LevelWarn, fmt.Sprintf(msg, args...), 0, nil)
}

func (l *Logger) Error(msg any, args ...attr.Attr) {
	l.log(context.Background(), LevelError, msg, 0, args)
}

func (l *Logger) Errorf(msg string, args ...any) {
	utils.EmulatePrintf(msg, args...)
	l.log(context.Background(), LevelError, fmt.Sprintf(msg, args...), 0, nil)
}

func (l *Logger) ErrorContext(ctx context.Context, msg any, args ...attr.Attr) {
	l.log(ctx, LevelError, msg, 0, args)
}

func (l *Logger) ErrorfContext(ctx context.Context, msg string, args ...any) {
	utils.EmulatePrintf(msg, args...)
	l.log(ctx, LevelError, fmt.Sprintf(msg, args...), 0, nil)
}

func (l *Logger) Fatal(msg any, args ...attr.Attr) {
	l.log(context.Background(), LevelFatal, msg, 0, args)
}

func (l *Logger) Fatalf(msg string, args ...any) {
	utils.EmulatePrintf(msg, args...)
	l.log(context.Background(), LevelFatal, fmt.Sprintf(msg, args...), 0, nil)
}

func (l *Logger) FatalContext(ctx context.Context, msg any, args ...attr.Attr) {
	l.log(ctx, LevelFatal, msg, 0, args)
}

func (l *Logger) FatalfContext(ctx context.Context, msg string, args ...any) {
	utils.EmulatePrintf(msg, args...)
	l.log(ctx, LevelFatal, fmt.Sprintf(msg, args...), 0, nil)
}

func (l *Logger) Panic(err error, args ...attr.Attr) {
	l.log(context.Background(), LevelFatal, err, 0, args)
	panic(err)
}

func (l *Logger) Panicf(msg string, args ...any) {
	utils.EmulateErrorf(msg, args...)
	//nolint:goerr113
	err := fmt.Errorf(msg, args...)
	l.log(context.Background(), LevelFatal, err, 0, nil)
	panic(err)
}

func (l *Logger) PanicContext(ctx context.Context, err error, args ...attr.Attr) {
	l.log(ctx, LevelFatal, err, 0, args)
	panic(err)
}

func (l *Logger) PanicfContext(ctx context.Context, msg string, args ...any) {
	utils.EmulateErrorf(msg, args...)
	//nolint:goerr113
	err := fmt.Errorf(msg, args...)
	l.log(ctx, LevelFatal, err, 0, nil)
	panic(err)
}

func (l *Logger) Log(ctx context.Context, level Level, msg any, args ...attr.Attr) {
	l.log(ctx, level, msg, 0, args)
}

func (l *Logger) LogHelper(ctx context.Context, level Level, msg any, args ...attr.Attr) {
	l.log(ctx, level, msg, 1, args)
}

const numSkipFrames = 3

func (l *Logger) log(ctx context.Context, level Level, msg any, skip int, args []attr.Attr) {
	if !l.logger.Enabled(ctx, slog.Level(level)) {
		return
	}

	var message string
	if val, ok := msg.(string); ok {
		message = val
	} else {
		message = lit.Sdump(msg)
	}

	var pcs [1]uintptr

	runtime.Callers(numSkipFrames+skip, pcs[:])
	r := slog.NewRecord(time.Now(), slog.Level(level), message, pcs[0])
	r.AddAttrs(args...)
	_ = l.logger.Handler().Handle(ctx, r)
}
