package attr_test

import (
	"errors"
	"testing"
	"time"

	"github.com/DimmyJing/valise/attr"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
)

type mockStringer struct{}

func (m mockStringer) String() string { return "mockStringer" }

type mockLogValuer struct{}

func (m mockLogValuer) LogValue() attr.Value { return attr.StringValue("mockLogValuer") }

type mockJSONError struct{}

var errMockJSON = errors.New("mockJSONError")

func (m mockJSONError) MarshalJSON() ([]byte, error) { return nil, errMockJSON }

func TestOtelValue(t *testing.T) { //nolint:funlen
	t.Parallel()
	assert := assert.New(t)
	assert.Equal(attribute.Int64Value(1), attr.AnyToOtelValue(int8(1)))
	assert.Equal(attribute.Int64Value(1), attr.AnyToOtelValue(int16(1)))
	assert.Equal(attribute.Int64Value(1), attr.AnyToOtelValue(int32(1)))
	assert.Equal(attribute.Int64Value(1), attr.AnyToOtelValue(int64(1)))
	assert.Equal(attribute.Int64Value(1), attr.AnyToOtelValue(int(1)))
	assert.Equal(attribute.Int64Value(1), attr.AnyToOtelValue(uint8(1)))
	assert.Equal(attribute.Int64Value(1), attr.AnyToOtelValue(uint16(1)))
	assert.Equal(attribute.Int64Value(1), attr.AnyToOtelValue(uint32(1)))
	assert.Equal(attribute.Int64Value(1), attr.AnyToOtelValue(uint64(1)))
	assert.Equal(attribute.Int64Value(1), attr.AnyToOtelValue(uint(1)))
	assert.Equal(attribute.Int64Value(1), attr.AnyToOtelValue(uintptr(1)))
	assert.Equal(attribute.Float64Value(1.5), attr.AnyToOtelValue(float32(1.5)))
	assert.Equal(attribute.Float64Value(1.5), attr.AnyToOtelValue(float64(1.5)))
	assert.Equal(attribute.BoolValue(true), attr.AnyToOtelValue(true))
	assert.Equal(attribute.StringValue("hello"), attr.AnyToOtelValue("hello"))
	assert.Equal(attribute.Int64SliceValue([]int64{1, 2}), attr.AnyToOtelValue([]int8{1, 2}))
	assert.Equal(attribute.Int64SliceValue([]int64{1, 2}), attr.AnyToOtelValue([]int16{1, 2}))
	assert.Equal(attribute.Int64SliceValue([]int64{1, 2}), attr.AnyToOtelValue([]int32{1, 2}))
	assert.Equal(attribute.Int64SliceValue([]int64{1, 2}), attr.AnyToOtelValue([]int64{1, 2}))
	assert.Equal(attribute.Int64SliceValue([]int64{1, 2}), attr.AnyToOtelValue([]int{1, 2}))
	assert.Equal(attribute.Int64SliceValue([]int64{1, 2}), attr.AnyToOtelValue([]uint8{1, 2}))
	assert.Equal(attribute.Int64SliceValue([]int64{1, 2}), attr.AnyToOtelValue([]uint16{1, 2}))
	assert.Equal(attribute.Int64SliceValue([]int64{1, 2}), attr.AnyToOtelValue([]uint32{1, 2}))
	assert.Equal(attribute.Int64SliceValue([]int64{1, 2}), attr.AnyToOtelValue([]uint64{1, 2}))
	assert.Equal(attribute.Int64SliceValue([]int64{1, 2}), attr.AnyToOtelValue([]uint{1, 2}))
	assert.Equal(attribute.Int64SliceValue([]int64{1, 2}), attr.AnyToOtelValue([]uintptr{1, 2}))
	assert.Equal(attribute.Float64SliceValue([]float64{1.5, 2.5}), attr.AnyToOtelValue([]float32{1.5, 2.5}))
	assert.Equal(attribute.Float64SliceValue([]float64{1.5, 2.5}), attr.AnyToOtelValue([]float64{1.5, 2.5}))
	assert.Equal(attribute.BoolSliceValue([]bool{true, false}), attr.AnyToOtelValue([]bool{true, false}))
	assert.Equal(attribute.StringSliceValue([]string{"hello", "world"}), attr.AnyToOtelValue([]string{"hello", "world"}))
	assert.Equal(attribute.Int64SliceValue([]int64{1, 2}), attr.AnyToOtelValue([]any{int8(1), int8(2)}))
	assert.Equal(attribute.Int64SliceValue([]int64{1, 2}), attr.AnyToOtelValue([]any{int16(1), int16(2)}))
	assert.Equal(attribute.Int64SliceValue([]int64{1, 2}), attr.AnyToOtelValue([]any{int32(1), int32(2)}))
	assert.Equal(attribute.Int64SliceValue([]int64{1, 2}), attr.AnyToOtelValue([]any{int64(1), int64(2)}))
	assert.Equal(attribute.Int64SliceValue([]int64{1, 2}), attr.AnyToOtelValue([]any{int(1), int(2)}))
	assert.Equal(attribute.Int64SliceValue([]int64{1, 2}), attr.AnyToOtelValue([]any{uint8(1), uint8(2)}))
	assert.Equal(attribute.Int64SliceValue([]int64{1, 2}), attr.AnyToOtelValue([]any{uint16(1), uint16(2)}))
	assert.Equal(attribute.Int64SliceValue([]int64{1, 2}), attr.AnyToOtelValue([]any{uint32(1), uint32(2)}))
	assert.Equal(attribute.Int64SliceValue([]int64{1, 2}), attr.AnyToOtelValue([]any{uint64(1), uint64(2)}))
	assert.Equal(attribute.Int64SliceValue([]int64{1, 2}), attr.AnyToOtelValue([]any{uint(1), uint(2)}))
	assert.Equal(attribute.Int64SliceValue([]int64{1, 2}), attr.AnyToOtelValue([]any{uintptr(1), uintptr(2)}))
	assert.Equal(attribute.Float64SliceValue([]float64{1.5, 2.5}), attr.AnyToOtelValue([]any{float32(1.5), float32(2.5)}))
	assert.Equal(attribute.Float64SliceValue([]float64{1.5, 2.5}), attr.AnyToOtelValue([]any{float64(1.5), float64(2.5)}))
	assert.Equal(attribute.BoolSliceValue([]bool{true, false}), attr.AnyToOtelValue([]any{true, false}))
	assert.Equal(attribute.StringSliceValue([]string{"hello", "world"}), attr.AnyToOtelValue([]any{"hello", "world"}))
	assert.Equal(attribute.StringSliceValue([]string{"1", "1.5"}), attr.AnyToOtelValue([]any{1, 1.5}))
	assert.Equal(attribute.StringValue("mockStringer"), attr.AnyToOtelValue(mockStringer{}))
	assert.Equal(attribute.StringValue("hello"), attr.AnyToOtelValue(attr.StringValue("hello")))
	assert.Equal(attribute.StringValue("hello"), attr.AnyToOtelValue(attribute.StringValue("hello")))
	assert.Equal(attribute.StringValue("mockLogValuer"), attr.AnyToOtelValue(mockLogValuer{}))
	assert.Equal(
		attribute.StringValue(
			"failed to marshal attribute: json: error calling MarshalJSON for type attr_test.mockJSONError: mockJSONError",
		),
		attr.AnyToOtelValue(mockJSONError{}),
	)
	assert.Equal(
		attribute.StringSliceValue(
			[]string{"1970-01-01T00:00:00Z", "1970-01-01T00:00:01.000000001Z"},
		),
		attr.AnyToOtelValue([]time.Time{time.Unix(0, 0).UTC(), time.Unix(1, 1).UTC()}),
	)
	assert.Equal(
		attribute.StringValue("{\"key1\":\"val1\",\"key2\":\"val2\"}"),
		attr.AnyToOtelValue(attr.GroupValue(attr.String("key2", "val2"), attr.String("key1", "val1"))),
	)
	assert.Equal(attribute.StringValue("hello"), attr.OtelValue(attr.StringValue("hello")))
}
