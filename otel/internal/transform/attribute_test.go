package transform_test

import (
	"testing"

	"github.com/DimmyJing/valise/otel/internal/transform"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/otel/attribute"
)

func testTransform(t *testing.T, attr attribute.Value, val pcommon.Value) {
	t.Helper()

	testMap := pcommon.NewMap()
	val.CopyTo(testMap.PutEmpty("testkey"))

	pMap := pcommon.NewMap()
	transform.TransformKeyValues([]attribute.KeyValue{
		{Key: "testkey", Value: attr},
	}, pMap)

	assert.Equal(t, testMap, pMap)
}

func mustFromRaw(t *testing.T, val pcommon.Value, from any) {
	t.Helper()

	err := val.FromRaw(from)
	if err != nil {
		t.Fatal(err)
	}
}

func TestTransformKeyValue(t *testing.T) {
	t.Parallel()

	testTransform(t, attribute.BoolValue(true), pcommon.NewValueBool(true))
	testTransform(t, attribute.Int64Value(123), pcommon.NewValueInt(123))
	testTransform(t, attribute.Float64Value(1.2), pcommon.NewValueDouble(1.2))
	testTransform(t, attribute.StringValue("test"), pcommon.NewValueStr("test"))
	testTransform(t, attribute.Value{}, pcommon.NewValueStr("INVALID"))

	val := pcommon.NewValueEmpty()
	mustFromRaw(t, val, []any{true, false})
	testTransform(t, attribute.BoolSliceValue([]bool{true, false}), val)
	mustFromRaw(t, val, []any{1, 2})
	testTransform(t, attribute.Int64SliceValue([]int64{1, 2}), val)
	mustFromRaw(t, val, []any{1.5, 2.5})
	testTransform(t, attribute.Float64SliceValue([]float64{1.5, 2.5}), val)
	mustFromRaw(t, val, []any{"test1", "test2"})
	testTransform(t, attribute.StringSliceValue([]string{"test1", "test2"}), val)
}
