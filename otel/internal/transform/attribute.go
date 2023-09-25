package transform

import (
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/otel/attribute"
)

func transformKeyValues(attrs []attribute.KeyValue, pMap pcommon.Map) {
	l := len(attrs)
	if l == 0 {
		return
	}

	pMap.EnsureCapacity(l)

	for _, kv := range attrs {
		transformValue(kv.Value, pMap.PutEmpty(string(kv.Key)))
	}
}

func transformAttrIter(iter attribute.Iterator, pMap pcommon.Map) {
	l := iter.Len()
	if l == 0 {
		return
	}

	pMap.EnsureCapacity(l)

	for iter.Next() {
		attr := iter.Attribute()
		transformValue(attr.Value, pMap.PutEmpty(string(attr.Key)))
	}
}

func transformValue(val attribute.Value, pVal pcommon.Value) { //nolint:cyclop
	switch val.Type() {
	case attribute.BOOL:
		pVal.SetBool(val.AsBool())
	case attribute.BOOLSLICE:
		boolSliceValues(val.AsBoolSlice(), pVal)
	case attribute.INT64:
		pVal.SetInt(val.AsInt64())
	case attribute.INT64SLICE:
		int64SliceValues(val.AsInt64Slice(), pVal)
	case attribute.FLOAT64:
		pVal.SetDouble(val.AsFloat64())
	case attribute.FLOAT64SLICE:
		float64SliceValues(val.AsFloat64Slice(), pVal)
	case attribute.STRING:
		pVal.SetStr(val.AsString())
	case attribute.STRINGSLICE:
		stringSliceValues(val.AsStringSlice(), pVal)
	case attribute.INVALID:
		pVal.SetStr("INVALID")
	default:
		pVal.SetStr("INVALID")
	}
}

func boolSliceValues(vals []bool, val pcommon.Value) {
	val.SetEmptySlice()
	slice := val.Slice()
	slice.EnsureCapacity(len(vals))

	for _, v := range vals {
		slice.AppendEmpty().SetBool(v)
	}
}

func int64SliceValues(vals []int64, val pcommon.Value) {
	val.SetEmptySlice()
	slice := val.Slice()
	slice.EnsureCapacity(len(vals))

	for _, v := range vals {
		slice.AppendEmpty().SetInt(v)
	}
}

func float64SliceValues(vals []float64, val pcommon.Value) {
	val.SetEmptySlice()
	slice := val.Slice()
	slice.EnsureCapacity(len(vals))

	for _, v := range vals {
		slice.AppendEmpty().SetDouble(v)
	}
}

func stringSliceValues(vals []string, val pcommon.Value) {
	val.SetEmptySlice()
	slice := val.Slice()
	slice.EnsureCapacity(len(vals))

	for _, v := range vals {
		slice.AppendEmpty().SetStr(v)
	}
}
