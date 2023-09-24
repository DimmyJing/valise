package attr

import (
	"log/slog"
	"time"
)

type Attr = slog.Attr

func String(key string, value string) Attr {
	return Attr{Key: key, Value: StringValue(value)}
}

func Int64(key string, value int64) Attr {
	return Attr{Key: key, Value: Int64Value(value)}
}

func Int(key string, value int) Attr {
	return Attr{Key: key, Value: IntValue(value)}
}

func Uint64(key string, value uint64) Attr {
	return Attr{Key: key, Value: Uint64Value(value)}
}

func Float64(key string, value float64) Attr {
	return Attr{Key: key, Value: Float64Value(value)}
}

func Bool(key string, value bool) Attr {
	return Attr{Key: key, Value: BoolValue(value)}
}

func Time(key string, value time.Time) Attr {
	return Attr{Key: key, Value: TimeValue(value)}
}

func Duration(key string, value time.Duration) Attr {
	return Attr{Key: key, Value: DurationValue(value)}
}

func Group(key string, value ...Attr) Attr {
	return Attr{Key: key, Value: GroupValue(value...)}
}

func Any(key string, value any) Attr {
	return Attr{Key: key, Value: AnyValue(value)}
}
