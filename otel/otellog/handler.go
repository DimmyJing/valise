package otellog

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/DimmyJing/valise/attr"
	"github.com/DimmyJing/valise/log"
	"go.opentelemetry.io/otel/attribute"
)

func recordAttrs(r slog.Record) []attribute.KeyValue {
	res := make([]attribute.KeyValue, 0, r.NumAttrs())
	r.Attrs(func(a slog.Attr) bool {
		res = append(res, attr.OtelAttr(a))

		return true
	})

	return res
}

func GetLogHandler(logger Logger) log.Option { //nolint:ireturn
	return log.WithHandler(func(ctx context.Context, record slog.Record) error {
		err := logger.Log(
			ctx,
			record.Level.String(),
			SeverityNumber(record.Level),
			attribute.StringValue(record.Message),
			recordAttrs(record),
		)
		if err != nil {
			return fmt.Errorf("failed to export logs: %w", err)
		}

		return nil
	})
}
