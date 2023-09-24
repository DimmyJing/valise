package attr_test

import (
	"testing"
	"time"

	"github.com/DimmyJing/valise/attr"
	"github.com/stretchr/testify/assert"
)

func TestString(t *testing.T) {
	t.Parallel()

	a := attr.String("foo", "bar")
	assert.Equal(t, "foo", a.Key)
	assert.Equal(t, "bar", a.Value.String())
}

func TestInt64(t *testing.T) {
	t.Parallel()

	a := attr.Int64("foo", 42)
	assert.Equal(t, "foo", a.Key)
	assert.Equal(t, int64(42), a.Value.Int64())
}

func TestInt(t *testing.T) {
	t.Parallel()

	a := attr.Int("foo", 42)
	assert.Equal(t, "foo", a.Key)
	assert.Equal(t, int64(42), a.Value.Int64())
}

func TestUint64(t *testing.T) {
	t.Parallel()

	a := attr.Uint64("foo", 42)
	assert.Equal(t, "foo", a.Key)
	assert.Equal(t, uint64(42), a.Value.Uint64())
}

func TestFloat64(t *testing.T) {
	t.Parallel()

	a := attr.Float64("foo", 42.0)
	assert.Equal(t, "foo", a.Key)
	assert.Equal(t, 42.0, a.Value.Float64())
}

func TestBool(t *testing.T) {
	t.Parallel()

	a := attr.Bool("foo", true)
	assert.Equal(t, "foo", a.Key)
	assert.Equal(t, true, a.Value.Bool())
}

func TestTime(t *testing.T) {
	t.Parallel()

	now := time.Now()
	att := attr.Time("foo", now)
	assert.Equal(t, "foo", att.Key)
	assert.Equal(t, now.Round(0), att.Value.Time().Round(0))

	att = attr.Time("foo", time.Time{})
	assert.Equal(t, "foo", att.Key)
	assert.Equal(t, time.Time{}, att.Value.Time())
}

func TestDuration(t *testing.T) {
	t.Parallel()

	att := attr.Duration("foo", time.Second)
	assert.Equal(t, "foo", att.Key)
	assert.Equal(t, time.Second, att.Value.Duration())
}

func TestGroup(t *testing.T) {
	t.Parallel()

	att := attr.Group("foo", attr.String("bar", "baz"))
	assert.Equal(t, "foo", att.Key)
	assert.Equal(t, "bar", att.Value.Group()[0].Key)
	assert.Equal(t, "baz", att.Value.Group()[0].Value.String())
}

func TestAny(t *testing.T) {
	t.Parallel()

	att := attr.Any("foo", "bar")
	assert.Equal(t, "foo", att.Key)
	assert.Equal(t, "bar", att.Value.Any())
}
