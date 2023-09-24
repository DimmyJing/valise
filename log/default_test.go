package log_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/DimmyJing/valise/attr"
	"github.com/DimmyJing/valise/log"
	"github.com/stretchr/testify/assert"
)

func initDefault(options ...log.Option) *bytes.Buffer {
	buf := new(bytes.Buffer)
	options = append(options, log.WithWriter(buf))
	l := log.New(options...)
	log.SetDefault(l)

	return buf
}

func TestDefaultLogger(t *testing.T) { //nolint:paralleltest
	buf := initDefault(log.WithCharm())
	log.Default().Error("testdefault")
	log.With(attr.String("key", "testwith")).Warn("")

	bufStr := buf.String()

	assert.Contains(t, bufStr, "ERRO")
	assert.Contains(t, bufStr, "testdefault")
	assert.Contains(t, bufStr, "WARN")
	assert.Contains(t, bufStr, "testwith")
}

func TestDefaultTrace(t *testing.T) { //nolint:paralleltest
	buf := initDefault(log.WithLogLevel(log.LevelAll))
	log.Trace("testtrace1")
	log.Tracef("%s", "testtrace2")
	log.TraceContext(context.Background(), "testtrace3")
	log.TracefContext(context.Background(), "%s", "testtrace4")

	bufStr := buf.String()
	for _, line := range strings.Split(strings.TrimSpace(bufStr), "\n") {
		assert.Contains(t, line, "TRACE")
		assert.Contains(t, line, "testtrace")
	}
}

func TestDefaultDebug(t *testing.T) { //nolint:paralleltest
	buf := initDefault(log.WithLogLevel(log.LevelAll))
	log.Debug("testdebug1")
	log.Debugf("%s", "testdebug2")
	log.DebugContext(context.Background(), "testdebug3")
	log.DebugfContext(context.Background(), "%s", "testdebug4")

	bufStr := buf.String()
	for _, line := range strings.Split(strings.TrimSpace(bufStr), "\n") {
		assert.Contains(t, line, "DEBUG")
		assert.Contains(t, line, "testdebug")
	}
}

func TestDefaultInfo(t *testing.T) { //nolint:paralleltest
	buf := initDefault(log.WithLogLevel(log.LevelAll))
	log.Info("testinfo1")
	log.Infof("%s", "testinfo2")
	log.InfoContext(context.Background(), "testinfo3")
	log.InfofContext(context.Background(), "%s", "testinfo4")

	bufStr := buf.String()
	for _, line := range strings.Split(strings.TrimSpace(bufStr), "\n") {
		assert.Contains(t, line, "INFO")
		assert.Contains(t, line, "testinfo")
	}
}

func TestDefaultWarn(t *testing.T) { //nolint:paralleltest
	buf := initDefault(log.WithLogLevel(log.LevelAll))
	log.Warn("testwarn1")
	log.Warnf("%s", "testwarn2")
	log.WarnContext(context.Background(), "testwarn3")
	log.WarnfContext(context.Background(), "%s", "testwarn4")

	bufStr := buf.String()
	for _, line := range strings.Split(strings.TrimSpace(bufStr), "\n") {
		assert.Contains(t, line, "WARN")
		assert.Contains(t, line, "testwarn")
	}
}

func TestDefaultError(t *testing.T) { //nolint:paralleltest
	buf := initDefault(log.WithLogLevel(log.LevelAll))
	log.Error("testerror1")
	log.Errorf("%s", "testerror2")
	log.ErrorContext(context.Background(), "testerror3")
	log.ErrorfContext(context.Background(), "%s", "testerror4")

	bufStr := buf.String()
	for _, line := range strings.Split(strings.TrimSpace(bufStr), "\n") {
		assert.Contains(t, line, "ERROR")
		assert.Contains(t, line, "testerror")
	}
}

func TestDefaultFatal(t *testing.T) { //nolint:paralleltest
	buf := initDefault(log.WithLogLevel(log.LevelAll))
	log.Fatal("testfatal1")
	log.Fatalf("%s", "testfatal2")
	log.FatalContext(context.Background(), "testfatal3")
	log.FatalfContext(context.Background(), "%s", "testfatal4")

	bufStr := buf.String()
	for _, line := range strings.Split(strings.TrimSpace(bufStr), "\n") {
		assert.Contains(t, line, "FATAL")
		assert.Contains(t, line, "testfatal")
	}
}

func TestDefaultPanic(t *testing.T) { //nolint:paralleltest
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

	buf := initDefault(log.WithLogLevel(log.LevelAll))

	testPanic(func() {
		log.Panic(errPanic)
	})
	testPanic(func() {
		log.Panicf("%w", errPanic)
	})
	testPanic(func() {
		log.PanicContext(context.Background(), errPanic)
	})
	testPanic(func() {
		log.PanicfContext(context.Background(), "%w", errPanic)
	})

	bufStr := buf.String()
	for _, line := range strings.Split(strings.TrimSpace(bufStr), "\n") {
		assert.Contains(t, line, "FATAL")
		assert.Contains(t, line, "testpanic")
	}
}
