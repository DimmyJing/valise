package attr

import (
	"log/slog"
	"time"
)

type (
	Value     = slog.Value
	Kind      = slog.Kind
	LogValuer = slog.LogValuer
)

func StringValue(val string) Value {
	return slog.StringValue(val)
}

func IntValue(val int) Value {
	return slog.IntValue(val)
}

func Int64Value(val int64) Value {
	return slog.Int64Value(val)
}

func Uint64Value(val uint64) Value {
	return slog.Uint64Value(val)
}

func Float64Value(val float64) Value {
	return slog.Float64Value(val)
}

func BoolValue(val bool) Value {
	return slog.BoolValue(val)
}

func TimeValue(val time.Time) Value {
	return slog.TimeValue(val)
}

func DurationValue(val time.Duration) Value {
	return slog.DurationValue(val)
}

func GroupValue(as ...Attr) Value {
	return slog.GroupValue(as...)
}

func AnyValue(val any) Value {
	return slog.AnyValue(val)
}

const maxLogValues = 100

func ToAny(val Value) any {
	newVal := val.Any()
	for i := 0; i < maxLogValues; i++ {
		if logValuer, ok := newVal.(LogValuer); ok {
			newVal = logValuer.LogValue().Any()
		} else {
			break
		}
	}

	if group, ok := newVal.([]Attr); ok {
		res := make(map[string]any)
		for _, a := range group {
			res[a.Key] = ToAny(a.Value)
		}

		return res
	}

	return newVal
}
