package log_test

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/DimmyJing/valise/attr"
	"github.com/DimmyJing/valise/log"
	"github.com/stretchr/testify/assert"
)

func TestWithCharmHandler(t *testing.T) {
	t.Parallel()

	logger, buf := getLogger(log.WithCharm())
	logger.Info("test")

	bufStr := buf.String()
	assert.NotContains(t, bufStr, "{")
	assert.Contains(t, bufStr, "test")
	assert.Contains(t, bufStr, "INFO")
}

func TestWithLevelHandler(t *testing.T) {
	t.Parallel()

	logger, buf := getLogger(
		log.WithLevel(log.LevelOff),
		log.WithHandler(func(ctx context.Context, record slog.Record) error {
			t.Fail()

			return nil
		}),
	)
	logger.Info("test")
	assert.Empty(t, strings.TrimSpace(buf.String()))
}

func TestWithLogLevelHandler(t *testing.T) {
	t.Parallel()

	res := false
	logger, buf := getLogger(
		log.WithLogLevel(log.LevelOff),
		log.WithHandler(func(ctx context.Context, record slog.Record) error {
			res = true

			return nil
		}),
	)
	logger.Info("test")
	assert.Empty(t, strings.TrimSpace(buf.String()))
	assert.True(t, res)
}

func TestWithNoSourceHandler(t *testing.T) {
	t.Parallel()

	logger, buf := getLogger(log.WithNoSource())
	logger.Info("test")
	assert.NotContains(t, buf.String(), "source")
}

func TestWithReplaceAttrHandler(t *testing.T) {
	t.Parallel()

	logger, buf := getLogger(log.WithReplaceAttr(func(groups []string, a attr.Attr) attr.Attr {
		if a.Key == "testkey" {
			return attr.String("testkey", "testvalue1")
		}

		return a
	}))
	logger.Info(
		"test",
		attr.String("testkey", "placeholder"),
		attr.String("testkey2", "testvalue2"),
	)

	bufStr := buf.String()
	assert.Contains(t, bufStr, "testkey")
	assert.Contains(t, bufStr, "testvalue1")
	assert.Contains(t, bufStr, "testkey2")
	assert.Contains(t, bufStr, "testvalue2")
	assert.NotContains(t, bufStr, "placeholder")
}

func TestCharmLevel(t *testing.T) {
	t.Parallel()

	logger, _ := getLogger(log.WithCharm(), log.WithLogLevel(log.LevelAll))
	logger.Debug("test")
	logger.Info("test")
	logger.Warn("test")
	logger.Error("test")
	logger.Fatal("test")
}

func TestJSONHandlerGroup(t *testing.T) {
	t.Parallel()

	logger, buf := getLogger()
	logger.WithGroup("groupname").
		With(attr.String("key", "value")).
		Info("hello", attr.String("foo", "bar"))

	bufStr := buf.String()
	assert.Contains(t, bufStr, "groupname")
	assert.Contains(t, bufStr, "key")
	assert.Contains(t, bufStr, "value")
	assert.Contains(t, bufStr, "foo")
	assert.Contains(t, bufStr, "bar")
	assert.Contains(t, bufStr, "hello")
}

func TestHandlerGroup(t *testing.T) {
	t.Parallel()

	logger, buf := getLogger(log.WithCharm())
	logger.With(attr.String("key1", "val1")).
		WithGroup("groupname1").
		With(attr.String("key2", "val2")).
		WithGroup("groupname2").
		With(attr.String("key3", "val3")).
		Info("msg", attr.String("key4", "val4"))

	bufStr := buf.String()
	assert.Contains(t, bufStr, "groupname1")
	assert.Contains(t, bufStr, "groupname2")
	assert.Contains(t, bufStr, "key1")
	assert.Contains(t, bufStr, "key2")
	assert.Contains(t, bufStr, "key3")
	assert.Contains(t, bufStr, "key4")
	assert.Contains(t, bufStr, "val1")
	assert.Contains(t, bufStr, "val2")
	assert.Contains(t, bufStr, "val3")
	assert.Contains(t, bufStr, "val4")
}

func TestNormalCharm(t *testing.T) {
	t.Parallel()

	logger, buf := getLogger(log.WithCharm())
	logger.Info("test", attr.String("key", "value"))

	bufStr := buf.String()
	assert.Contains(t, bufStr, "INFO")
	assert.Contains(t, bufStr, "test")
	assert.Contains(t, bufStr, "key")
	assert.Contains(t, bufStr, "value")
}

type failWriter struct{}

func (w *failWriter) Write(p []byte) (int, error) {
	return 0, io.ErrClosedPipe
}

func TestWriterFail(t *testing.T) {
	t.Parallel()

	logger := log.New(log.WithWriter(&failWriter{}))
	logger.Info("test")
}

func TestHandlerFail(t *testing.T) {
	t.Parallel()

	logger, _ := getLogger(log.WithHandler(func(ctx context.Context, record slog.Record) error {
		return io.ErrClosedPipe
	}))
	logger.Info("test")
}

func TestHandlerZeroTime(t *testing.T) {
	t.Parallel()

	buf := new(bytes.Buffer)
	h := log.NewHandler(log.WithWriter(buf), log.WithCharm())
	err := h.Handle(context.Background(), slog.Record{
		Time:    time.Time{},
		Message: "test",
		Level:   slog.LevelInfo,
		PC:      uintptr(0),
	})
	assert.NoError(t, err)
}
