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

type Handler struct {
	writer      io.Writer
	useCharm    bool
	level       Leveler
	logLevel    Leveler
	addSource   bool
	replaceAttr func(groups []string, a attr.Attr) attr.Attr
	jsonHandler *slog.JSONHandler
	handler     func(ctx context.Context, record slog.Record) error
	charm       *log.Logger
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
		jsonHandler: nil,
		handler:     nil,
		charm:       nil,
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
		handler.charm = log.NewWithOptions(handler.writer, log.Options{
			ReportTimestamp: true,
			ReportCaller:    true,
			TimeFormat:      "15:04:05.000",
			Level:           log.DebugLevel,
		})
	} else {
		handler.jsonHandler = newJSONHandler(handler)
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
	//nolint:nestif
	if Level(record.Level) >= h.logLevel.Level() {
		if h.useCharm {
			err := h.charm.Handle(ctx, record)
			if err != nil {
				return fmt.Errorf("charm handler failed to handle log: %w", err)
			}
		} else {
			err := h.jsonHandler.Handle(ctx, record)
			if err != nil {
				return fmt.Errorf("json handler failed to handle log: %w", err)
			}
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
	newHandler := *h
	if h.useCharm {
		//nolint:forcetypeassert
		newHandler.charm = newHandler.charm.WithAttrs(attrs).(*log.Logger)
	} else {
		//nolint:forcetypeassert
		newHandler.jsonHandler = newHandler.jsonHandler.WithAttrs(attrs).(*slog.JSONHandler)
	}

	return &newHandler
}

func (h *Handler) WithGroup(name string) slog.Handler {
	newHandler := *h
	if h.useCharm {
		//nolint:forcetypeassert
		newHandler.charm = newHandler.charm.WithGroup(name).(*log.Logger)
	} else {
		//nolint:forcetypeassert
		newHandler.jsonHandler = newHandler.jsonHandler.WithGroup(name).(*slog.JSONHandler)
	}

	return &newHandler
}
