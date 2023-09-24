package log_test

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/DimmyJing/valise/attr"
	"github.com/DimmyJing/valise/log"
)

func TestWithCharm(t *testing.T) {
	t.Parallel()

	_ = log.WithCharm()
}

func TestWithLevel(t *testing.T) {
	t.Parallel()

	_ = log.WithLevel(log.LevelDebug)
}

func TestWithLogLevel(t *testing.T) {
	t.Parallel()

	_ = log.WithLogLevel(log.LevelInfo)
}

func TestWithNoSource(t *testing.T) {
	t.Parallel()

	_ = log.WithNoSource()
}

func TestWithReplaceAttr(t *testing.T) {
	t.Parallel()

	_ = log.WithReplaceAttr(func(_ []string, a attr.Attr) attr.Attr {
		return a
	})
}

func TestWithWriter(t *testing.T) {
	t.Parallel()

	_ = log.WithWriter(os.Stdout)
}

func TestWithHandler(t *testing.T) {
	t.Parallel()

	_ = log.WithHandler(func(_ context.Context, _ slog.Record) error {
		return nil
	})
}
