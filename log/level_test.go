package log_test

import (
	"testing"

	"github.com/DimmyJing/valise/log"
	"github.com/stretchr/testify/assert"
)

func TestLevelLevel(t *testing.T) {
	t.Parallel()

	var _ log.Leveler = log.LevelInfo

	assert.Equal(t, log.LevelError.Level(), log.LevelError)
}

func TestLevelMarshalJSON(t *testing.T) {
	t.Parallel()

	val, err := log.LevelError.MarshalJSON()

	assert.NoError(t, err)
	assert.Equal(t, []byte(`"ERROR"`), val)
}

func TestLevelMarshalText(t *testing.T) {
	t.Parallel()

	val, err := log.LevelError.MarshalText()

	assert.NoError(t, err)
	assert.Equal(t, []byte(`ERROR`), val)
}

func TestLevelString(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "ALL", log.LevelAll.String())
	assert.Equal(t, "TRACE", log.LevelTrace.String())
	assert.Equal(t, "DEBUG", log.LevelDebug.String())
	assert.Equal(t, "INFO", log.LevelInfo.String())
	assert.Equal(t, "WARN", log.LevelWarn.String())
	assert.Equal(t, "ERROR", log.LevelError.String())
	assert.Equal(t, "FATAL", log.LevelFatal.String())
	assert.Equal(t, "OFF", log.LevelOff.String())
	assert.Equal(t, "UNKNOWN-100", log.Level(100).String())
}

func TestLevelUnmarshalJSON(t *testing.T) {
	t.Parallel()

	var level log.Level

	err := level.UnmarshalJSON([]byte(`"ERROR"`))

	assert.NoError(t, err)
	assert.Equal(t, log.LevelError, level)

	err = level.UnmarshalJSON([]byte("ERROR"))

	assert.Error(t, err)
}

func TestLevelUnmarshalText(t *testing.T) {
	t.Parallel()

	var level log.Level

	err := level.UnmarshalText([]byte("ERROR"))

	assert.NoError(t, err)
	assert.Equal(t, log.LevelError, level)

	err = level.UnmarshalText([]byte("UNKNOWN-100"))

	assert.NoError(t, err)
	assert.Equal(t, log.Level(100), level)

	err = level.UnmarshalText([]byte("UNKNOWN-a"))

	assert.Error(t, err)

	err = level.UnmarshalText([]byte("INVALID"))

	assert.Error(t, err)

	testcases := []struct {
		in  string
		out log.Level
	}{
		{"ALL", log.LevelAll},
		{"TRACE", log.LevelTrace},
		{"DEBUG", log.LevelDebug},
		{"INFO", log.LevelInfo},
		{"WARN", log.LevelWarn},
		{"ERROR", log.LevelError},
		{"FATAL", log.LevelFatal},
		{"OFF", log.LevelOff},
	}
	for _, testcase := range testcases {
		err = level.UnmarshalText([]byte(testcase.in))

		assert.NoError(t, err)
		assert.Equal(t, testcase.out, level)
	}
}

func TestLevelVar(t *testing.T) {
	t.Parallel()

	var (
		level log.LevelVar
		_     log.Leveler = &level
	)

	assert.Equal(t, log.LevelInfo, level.Level())

	level.Set(log.LevelError)

	assert.Equal(t, log.LevelError, level.Level())

	text, err := level.MarshalText()

	assert.NoError(t, err)
	assert.Equal(t, []byte("ERROR"), text)
	assert.Equal(t, "LevelVar(ERROR)", level.String())

	err = level.UnmarshalText([]byte("WARN"))

	assert.NoError(t, err)
	assert.Equal(t, log.LevelWarn, level.Level())

	err = level.UnmarshalText([]byte("INVALID"))

	assert.Error(t, err)
	assert.Equal(t, log.LevelWarn, level.Level())
}
