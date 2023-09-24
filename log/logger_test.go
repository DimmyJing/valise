package log_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/DimmyJing/valise/attr"
	"github.com/DimmyJing/valise/log"
	"github.com/sanity-io/litter"
	"github.com/stretchr/testify/assert"
)

func getLogger(options ...log.Option) (*log.Logger, *bytes.Buffer) {
	buf := new(bytes.Buffer)
	options = append(options, log.WithWriter(buf))
	l := log.New(options...)

	return l, buf
}

func TestSetLitter(t *testing.T) { //nolint:paralleltest
	//nolint:exhaustruct
	log.SetLitterOptions(litter.Options{
		HidePrivateFields: false,
	})
}

func TestDefaultLogger(t *testing.T) { //nolint:paralleltest
	logger, buf := getLogger(log.WithCharm())
	log.SetDefault(logger)
	log.Default().Error("testdefault")
	log.With(attr.String("key", "testwith")).Warn("")

	bufStr := buf.String()

	assert.Contains(t, bufStr, "ERRO")
	assert.Contains(t, bufStr, "testdefault")
	assert.Contains(t, bufStr, "WARN")
	assert.Contains(t, bufStr, "testwith")
}

func TestLoggerEnabled(t *testing.T) {
	t.Parallel()

	logger, _ := getLogger(log.WithLevel(log.LevelInfo), log.WithCharm())
	assert.False(t, logger.Enabled(context.Background(), log.LevelDebug))
	assert.True(t, logger.Enabled(context.Background(), log.LevelInfo))
	assert.True(t, logger.Enabled(context.Background(), log.LevelError))
	assert.False(t, logger.Handler().Enabled(context.Background(), slog.Level(log.LevelDebug)))
	assert.True(t, logger.Handler().Enabled(context.Background(), slog.Level(log.LevelInfo)))
	assert.True(t, logger.Handler().Enabled(context.Background(), slog.Level(log.LevelError)))
}

func TestLoggerGroup(t *testing.T) {
	t.Parallel()

	logger, buf := getLogger(log.WithLevel(log.LevelInfo), log.WithCharm())
	logger.WithGroup("groupname").Info("test", attr.String("key", "testgroup"))
	bufStr := buf.String()
	assert.Contains(t, bufStr, "INFO")
	assert.Contains(t, bufStr, "groupname")
	assert.Contains(t, bufStr, "testgroup")
}

func TestLitterFormat(t *testing.T) {
	t.Parallel()

	logger, buf := getLogger(log.WithCharm())
	email := "test@gmail.com"
	obj := struct {
		name  string
		email *string
	}{name: "jimmy", email: &email}
	logger.Info(obj)

	bufStr := buf.String()
	assert.Contains(t, bufStr, "INFO")
	assert.Contains(t, bufStr, "name")
	assert.Contains(t, bufStr, "jimmy")
	assert.Contains(t, bufStr, "email")
	assert.Contains(t, bufStr, "test@gmail.com")
}

func TestTrace(t *testing.T) {
	t.Parallel()

	logger, buf := getLogger(log.WithLogLevel(log.LevelAll))
	logger.Trace("testtrace1")
	logger.Tracef("%s", "testtrace2")
	logger.TraceContext(context.Background(), "testtrace3")
	logger.TracefContext(context.Background(), "%s", "testtrace4")

	bufStr := buf.String()
	for _, line := range strings.Split(strings.TrimSpace(bufStr), "\n") {
		assert.Contains(t, line, "TRACE")
		assert.Contains(t, line, "testtrace")
	}
}

func TestDebug(t *testing.T) {
	t.Parallel()

	logger, buf := getLogger(log.WithLogLevel(log.LevelAll))
	logger.Debug("testdebug1")
	logger.Debugf("%s", "testdebug2")
	logger.DebugContext(context.Background(), "testdebug3")
	logger.DebugfContext(context.Background(), "%s", "testdebug4")

	bufStr := buf.String()
	for _, line := range strings.Split(strings.TrimSpace(bufStr), "\n") {
		assert.Contains(t, line, "DEBUG")
		assert.Contains(t, line, "testdebug")
	}
}

func TestInfo(t *testing.T) {
	t.Parallel()

	logger, buf := getLogger(log.WithLogLevel(log.LevelAll))
	logger.Info("testinfo1")
	logger.Infof("%s", "testinfo2")
	logger.InfoContext(context.Background(), "testinfo3")
	logger.InfofContext(context.Background(), "%s", "testinfo4")

	bufStr := buf.String()
	for _, line := range strings.Split(strings.TrimSpace(bufStr), "\n") {
		assert.Contains(t, line, "INFO")
		assert.Contains(t, line, "testinfo")
	}
}

func TestWarn(t *testing.T) {
	t.Parallel()

	logger, buf := getLogger(log.WithLogLevel(log.LevelAll))
	logger.Warn("testwarn1")
	logger.Warnf("%s", "testwarn2")
	logger.WarnContext(context.Background(), "testwarn3")
	logger.WarnfContext(context.Background(), "%s", "testwarn4")

	bufStr := buf.String()
	for _, line := range strings.Split(strings.TrimSpace(bufStr), "\n") {
		assert.Contains(t, line, "WARN")
		assert.Contains(t, line, "testwarn")
	}
}

func TestError(t *testing.T) {
	t.Parallel()

	logger, buf := getLogger(log.WithLogLevel(log.LevelAll))
	logger.Error("testerror1")
	logger.Errorf("%s", "testerror2")
	logger.ErrorContext(context.Background(), "testerror3")
	logger.ErrorfContext(context.Background(), "%s", "testerror4")

	bufStr := buf.String()
	for _, line := range strings.Split(strings.TrimSpace(bufStr), "\n") {
		assert.Contains(t, line, "ERROR")
		assert.Contains(t, line, "testerror")
	}
}

func TestFatal(t *testing.T) {
	t.Parallel()

	logger, buf := getLogger(log.WithLogLevel(log.LevelAll))
	logger.Fatal("testfatal1")
	logger.Fatalf("%s", "testfatal2")
	logger.FatalContext(context.Background(), "testfatal3")
	logger.FatalfContext(context.Background(), "%s", "testfatal4")

	bufStr := buf.String()
	for _, line := range strings.Split(strings.TrimSpace(bufStr), "\n") {
		assert.Contains(t, line, "FATAL")
		assert.Contains(t, line, "testfatal")
	}
}

var errPanic = errors.New("testpanic")

func TestPanic(t *testing.T) {
	t.Parallel()

	testPanic := func(function func()) {
		defer func() {
			if r := recover(); r != nil {
				if err, ok := r.(error); ok {
					assert.ErrorIs(t, err, errPanic)
				} else {
					t.Fail()
				}
			} else {
				t.Fail()
			}
		}()
		function()
	}

	logger, buf := getLogger(log.WithLogLevel(log.LevelAll))

	testPanic(func() {
		logger.Panic(errPanic)
	})
	testPanic(func() {
		logger.Panicf("%w", errPanic)
	})
	testPanic(func() {
		logger.PanicContext(context.Background(), errPanic)
	})
	testPanic(func() {
		logger.PanicfContext(context.Background(), "%w", errPanic)
	})

	bufStr := buf.String()
	for _, line := range strings.Split(strings.TrimSpace(bufStr), "\n") {
		assert.Contains(t, line, "FATAL")
		assert.Contains(t, line, "testpanic")
	}
}

func TestLog(t *testing.T) {
	t.Parallel()

	logger, buf := getLogger(log.WithLogLevel(log.LevelAll))
	logger.Log(context.Background(), log.LevelWarn, "testlog")

	bufStr := buf.String()
	assert.Contains(t, bufStr, "WARN")
	assert.Contains(t, bufStr, "testlog")
}

func TestLogHelper(t *testing.T) {
	t.Parallel()

	logger, buf := getLogger(log.WithLogLevel(log.LevelAll))
	logger.LogHelper(context.Background(), log.LevelWarn, "testlog")

	bufStr := buf.String()
	assert.Contains(t, bufStr, "WARN")
	assert.Contains(t, bufStr, "testlog")
}
