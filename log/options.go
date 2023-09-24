package log

import (
	"context"
	"io"
	"log/slog"

	"github.com/DimmyJing/valise/attr"
)

type Option interface {
	option()
}

// with charm log (https://github.com/charmbracelet/log)

type withCharm struct{}

func (withCharm) option() {}

func WithCharm() withCharm {
	return withCharm{}
}

type withLevel struct {
	level Level
}

func (withLevel) option() {}

func WithLevel(level Level) withLevel {
	return withLevel{level: level}
}

type withLogLevel struct {
	level Level
}

func (withLogLevel) option() {}

func WithLogLevel(level Level) withLogLevel {
	return withLogLevel{level: level}
}

type withNoSource struct{}

func (withNoSource) option() {}

func WithNoSource() withNoSource {
	return withNoSource{}
}

type withReplaceAttr struct {
	replaceAttr func(groups []string, a attr.Attr) attr.Attr
}

func (withReplaceAttr) option() {}

func WithReplaceAttr(replaceAttr func(groups []string, a attr.Attr) attr.Attr) withReplaceAttr {
	return withReplaceAttr{replaceAttr: replaceAttr}
}

type withWriter struct {
	writer io.Writer
}

func (withWriter) option() {}

func WithWriter(writer io.Writer) withWriter {
	return withWriter{writer: writer}
}

type withHandler struct {
	handler func(ctx context.Context, record slog.Record) error
}

func (withHandler) option() {}

func WithHandler(handler func(ctx context.Context, record slog.Record) error) withHandler {
	return withHandler{handler: handler}
}
