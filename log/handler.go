package log

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/DimmyJing/valise/attr"
	"github.com/charmbracelet/log"
)

// TODO: Add opentelemetry logging support

type Handler struct {
	writer      io.Writer
	useCharm    bool
	level       Leveler
	logLevel    Leveler
	addSource   bool
	replaceAttr func(groups []string, a attr.Attr) attr.Attr
	handler     func(ctx context.Context, record slog.Record) error
	slogHandler slog.Handler
	attrs       [][]attr.Attr
	groups      []string
}

func NewHandler(options ...Option) *Handler {
	handler := &Handler{
		writer:      os.Stdout,
		useCharm:    false,
		level:       LevelAll,
		logLevel:    LevelInfo,
		addSource:   true,
		replaceAttr: nil,
		handler:     nil,
		slogHandler: nil,
		attrs:       nil,
		groups:      nil,
	}

	for _, option := range options {
		switch opt := option.(type) {
		case withCharm:
			handler.useCharm = true
		case withLevel:
			handler.level = opt.level
		case withLogLevel:
			handler.logLevel = opt.level
		case withNoSource:
			handler.addSource = false
		case withReplaceAttr:
			handler.replaceAttr = opt.replaceAttr
		case withWriter:
			handler.writer = opt.writer
		case withHandler:
			handler.handler = opt.handler
		}
	}

	if handler.useCharm {
		//nolint:exhaustruct
		handler.slogHandler = log.NewWithOptions(handler.writer, log.Options{
			ReportTimestamp: true,
			ReportCaller:    true,
			TimeFormat:      "15:04:05.000",
			Level:           log.DebugLevel,
		})
	} else {
		handler.slogHandler = newJSONHandler(handler)
	}

	return handler
}

func newJSONHandler(handler *Handler) *slog.JSONHandler {
	return slog.NewJSONHandler(handler.writer, &slog.HandlerOptions{
		AddSource: handler.addSource,
		Level:     nil,
		ReplaceAttr: func(groups []string, a attr.Attr) attr.Attr {
			processedAttr := a

			if a.Key == slog.LevelKey {
				levelValue := a.Value.Any()
				if level, ok := levelValue.(slog.Level); ok {
					processedAttr = attr.Any(slog.LevelKey, Level(level))
				}
			}

			if handler.replaceAttr != nil {
				return handler.replaceAttr(groups, processedAttr)
			}

			return processedAttr
		},
	})
}

func (h *Handler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= slog.Level(h.level.Level())
}

func (h *Handler) Handle(ctx context.Context, record slog.Record) error {
	if Level(record.Level) >= h.logLevel.Level() {
		err := h.slogHandler.Handle(ctx, record)
		if err != nil {
			return fmt.Errorf("handler failed to handle log: %w", err)
		}
	}

	if Level(record.Level) >= h.level.Level() && h.handler != nil {
		err := h.handler(ctx, record)
		if err != nil {
			return fmt.Errorf("log handler failed to handle log: %w", err)
		}
	}

	return nil
}

func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	r := *h
	r.slogHandler = r.slogHandler.WithAttrs(attrs)

	return &r
}

func (h *Handler) WithGroup(name string) slog.Handler {
	r := *h
	r.slogHandler = r.slogHandler.WithGroup(name)

	return &r
}
