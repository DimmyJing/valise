package log

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"runtime"
	"time"

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
			TimeFormat: "15:04:05.000",
			Level:      log.DebugLevel,
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

func levelToCharm(charm *log.Logger, level Level) func(any, ...any) {
	//nolint:exhaustive
	switch level {
	case LevelTrace, LevelDebug:
		return charm.Debug
	case LevelInfo:
		return charm.Info
	case LevelWarn:
		return charm.Warn
	default:
		return charm.Error
	}
}

func getSource(pc uintptr) *slog.Source {
	fs := runtime.CallersFrames([]uintptr{pc})
	f, _ := fs.Next()

	return &slog.Source{
		Function: f.Function,
		File:     f.File,
		Line:     f.Line,
	}
}

func (h *Handler) getCharmAttrs(record slog.Record) []any { //nolint:cyclop
	attrs := []any{}

	var prevAttr *attr.Attr

	for idx := len(h.groups); idx >= 0; idx-- {
		//nolint:nestif
		if idx > 0 {
			var newAttrs []attr.Attr
			if len(h.attrs) > idx {
				newAttrs = make([]attr.Attr, len(h.attrs[idx]))
				copy(newAttrs, h.attrs[idx])
			}

			if idx < len(h.attrs)-1 {
				if prevAttr != nil {
					//nolint:makezero
					newAttrs = append(newAttrs, *prevAttr)
				}
			} else {
				record.Attrs(func(a slog.Attr) bool {
					//nolint:makezero
					newAttrs = append(newAttrs, a)

					return true
				})
			}

			if len(newAttrs) > 0 {
				attrGroup := attr.Group(h.groups[idx-1], newAttrs...)
				prevAttr = &attrGroup
			}
		} else {
			if len(h.attrs) > 0 {
				for _, a := range h.attrs[idx] {
					attrs = append(attrs, a.Key, attr.ToAny(a.Value))
				}
			}
			if prevAttr != nil {
				attrs = append(attrs, prevAttr.Key, attr.ToAny(prevAttr.Value))
			}
		}
	}

	if len(h.groups) == 0 {
		record.Attrs(func(a slog.Attr) bool {
			attrs = append(attrs, a.Key, attr.ToAny(a.Value))

			return true
		})
	}

	return attrs
}

func (h *Handler) printCharm(record slog.Record) {
	charm := h.charm.With()
	if !record.Time.IsZero() {
		charm.SetTimeFunction(func() time.Time {
			return record.Time
		})
		charm.SetReportTimestamp(true)
	} else {
		charm.SetReportTimestamp(false)
	}

	if h.addSource {
		source := getSource(record.PC)
		sourceStr := log.ShortCallerFormatter(source.File, source.Line, source.Function)

		charm.SetReportCaller(true)
		charm.SetCallerFormatter(func(_ string, _ int, _ string) string {
			return sourceStr
		})
	}

	attrs := h.getCharmAttrs(record)

	levelToCharm(charm, Level(record.Level))(record.Message, attrs...)
}

func (h *Handler) Handle(ctx context.Context, record slog.Record) error {
	if Level(record.Level) >= h.logLevel.Level() {
		if h.useCharm {
			h.printCharm(record)
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

func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler { //nolint:ireturn
	newHandler := *h
	if h.useCharm {
		newHandler.attrs = make([][]attr.Attr, len(h.attrs))
		for i, a := range h.attrs {
			newHandler.attrs[i] = make([]attr.Attr, len(a))
			copy(newHandler.attrs[i], a)
		}

		if len(h.attrs) <= len(h.groups) {
			newHandler.attrs = append(newHandler.attrs, nil)
		}

		attrsIdx := len(newHandler.attrs) - 1
		newHandler.attrs[attrsIdx] = append(newHandler.attrs[attrsIdx], attrs...)
	} else {
		//nolint:forcetypeassert
		newHandler.jsonHandler = newHandler.jsonHandler.WithAttrs(attrs).(*slog.JSONHandler)
	}

	return &newHandler
}

func (h *Handler) WithGroup(name string) slog.Handler { //nolint:ireturn
	newHandler := *h
	if h.useCharm {
		newHandler.groups = make([]string, len(h.groups)+1)
		copy(newHandler.groups, h.groups)
		newHandler.groups[len(h.groups)] = name
	} else {
		//nolint:forcetypeassert
		newHandler.jsonHandler = newHandler.jsonHandler.WithGroup(name).(*slog.JSONHandler)
	}

	return &newHandler
}
