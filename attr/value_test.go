package attr_test

import (
	"testing"

	"github.com/DimmyJing/valise/attr"
	"github.com/stretchr/testify/assert"
)

type customValue struct{}

func (v *customValue) LogValue() attr.Value {
	return attr.StringValue("bar")
}

func TestToAny(t *testing.T) {
	t.Parallel()

	val := attr.AnyValue(&customValue{})
	assert.Equal(t, "bar", attr.ToAny(val))

	val = attr.GroupValue(
		attr.String("foo", "bar"),
		attr.Group("foobar", attr.String("barbaz", "foobaz")),
	)
	assert.Equal(t, map[string]any{
		"foo": "bar",
		"foobar": map[string]any{
			"barbaz": "foobaz",
		},
	}, attr.ToAny(val))
}
