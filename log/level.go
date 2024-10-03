package log

import (
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync/atomic"
)

type Level slog.Level

const (
	LevelAll   Level = -12
	LevelTrace Level = -8
	LevelDebug Level = -4
	LevelInfo  Level = 0
	LevelWarn  Level = 4
	LevelError Level = 8
	LevelFatal Level = 12
	LevelOff   Level = 16
)

func (l Level) Level() Level {
	return l
}

func (l Level) MarshalJSON() ([]byte, error) {
	return strconv.AppendQuote(nil, l.String()), nil
}

func (l Level) MarshalText() ([]byte, error) {
	return []byte(l.String()), nil
}

func (l Level) String() string {
	switch l {
	case LevelAll:
		return "ALL"
	case LevelTrace:
		return "TRACE"
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	case LevelFatal:
		return "FATAL"
	case LevelOff:
		return "OFF"
	default:
		return "UNKNOWN-" + strconv.FormatInt(int64(l), 10)
	}
}

func (l *Level) UnmarshalJSON(data []byte) error {
	s, err := strconv.Unquote(string(data))
	if err != nil {
		return fmt.Errorf("failed to unquote %q: %w", string(data), err)
	}

	return l.parse(s)
}

func (l *Level) UnmarshalText(data []byte) error {
	return l.parse(string(data))
}

var errInvalidLevel = errors.New("invalid level")

//nolint:cyclop
func (l *Level) parse(str string) error {
	switch str {
	case "ALL":
		*l = LevelAll
	case "TRACE":
		*l = LevelTrace
	case "DEBUG":
		*l = LevelDebug
	case "INFO":
		*l = LevelInfo
	case "WARN":
		*l = LevelWarn
	case "ERROR":
		*l = LevelError
	case "FATAL":
		*l = LevelFatal
	case "OFF":
		*l = LevelOff
	default:
		if cut, found := strings.CutPrefix(str, "UNKNOWN-"); found {
			res, err := strconv.ParseInt(cut, 10, 8)
			if err != nil {
				return fmt.Errorf("failed to parse int %q: %w", str, err)
			}

			*l = Level(res)
		} else {
			return fmt.Errorf("unknown level %q: %w", str, errInvalidLevel)
		}
	}

	return nil
}

type LevelVar struct {
	level atomic.Int32
}

func (l *LevelVar) Level() Level {
	return Level(l.level.Load())
}

func (l *LevelVar) MarshalText() ([]byte, error) {
	return l.Level().MarshalText()
}

func (l *LevelVar) Set(level Level) {
	//nolint:gosec
	l.level.Store(int32(level))
}

func (l *LevelVar) String() string {
	return fmt.Sprintf("LevelVar(%s)", l.Level())
}

func (l *LevelVar) UnmarshalText(data []byte) error {
	var level Level
	if err := level.UnmarshalText(data); err != nil {
		return err
	}

	l.Set(level)

	return nil
}

type Leveler interface {
	Level() Level
}
