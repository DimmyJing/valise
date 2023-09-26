package attr

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"reflect"
	"strconv"

	"go.opentelemetry.io/otel/attribute"
)

type integral interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr
}

func convertSliceIntegral[N integral](slice []N) attribute.Value {
	res := make([]int64, len(slice))

	for i, val := range slice {
		res[i] = int64(val)
	}

	return attribute.Int64SliceValue(res)
}

func convertAnySliceIntegral[N integral](slice []any) (attribute.Value, bool) {
	res := make([]int64, len(slice))

	for i, val := range slice {
		if val, ok := val.(N); ok {
			res[i] = int64(val)
		} else {
			return attribute.BoolValue(false), false
		}
	}

	return attribute.Int64SliceValue(res), true
}

func convertSliceFloat[N ~float32 | ~float64](slice []N) attribute.Value {
	res := make([]float64, len(slice))
	for i, val := range slice {
		res[i] = float64(val)
	}

	return attribute.Float64SliceValue(res)
}

func convertAnySliceFloat[N ~float32 | ~float64](slice []any) (attribute.Value, bool) {
	res := make([]float64, len(slice))

	for i, val := range slice {
		if val, ok := val.(N); ok {
			res[i] = float64(val)
		} else {
			return attribute.BoolValue(false), false
		}
	}

	return attribute.Float64SliceValue(res), true
}

func convertAnySlice[N any](slice []any) ([]N, bool) {
	res := make([]N, len(slice))

	for i, val := range slice {
		if val, ok := val.(N); ok {
			res[i] = val
		} else {
			return nil, false
		}
	}

	return res, true
}

func marshalJSON(val any) string {
	if res, err := json.Marshal(val); err != nil {
		return fmt.Sprintf("failed to marshal attribute: %v", err)
	} else {
		if unq, err := strconv.Unquote(string(res)); err != nil {
			return string(res)
		} else {
			return unq
		}
	}
}

func AnyToOtelValue(val any) attribute.Value { //nolint:funlen,gocognit,gocyclo,cyclop,maintidx
	switch val := val.(type) {
	case []slog.Attr:
		attrs := make([]attribute.KeyValue, len(val))
		for i, att := range val {
			attrs[i] = OtelAttr(att)
		}

		return AnyToOtelValue(attribute.NewSet(attrs...).MarshalLog())
	case attribute.Value:
		return val
	case LogValuer:
		return AnyToOtelValue(val.LogValue().Any())
	case Value:
		return AnyToOtelValue(val.Any())
	case int8:
		return attribute.Int64Value(int64(val))
	case int16:
		return attribute.Int64Value(int64(val))
	case int32:
		return attribute.Int64Value(int64(val))
	case int64:
		return attribute.Int64Value(val)
	case int:
		return attribute.Int64Value(int64(val))
	case uint8:
		return attribute.Int64Value(int64(val))
	case uint16:
		return attribute.Int64Value(int64(val))
	case uint32:
		return attribute.Int64Value(int64(val))
	case uint64:
		return attribute.Int64Value(int64(val))
	case uint:
		return attribute.Int64Value(int64(val))
	case uintptr:
		return attribute.Int64Value(int64(val))
	case float32:
		return attribute.Float64Value(float64(val))
	case float64:
		return attribute.Float64Value(val)
	case bool:
		return attribute.BoolValue(val)
	case string:
		return attribute.StringValue(val)
	case []int8:
		return convertSliceIntegral(val)
	case []int16:
		return convertSliceIntegral(val)
	case []int32:
		return convertSliceIntegral(val)
	case []int64:
		return attribute.Int64SliceValue(val)
	case []int:
		return convertSliceIntegral(val)
	case []uint8:
		return convertSliceIntegral(val)
	case []uint16:
		return convertSliceIntegral(val)
	case []uint32:
		return convertSliceIntegral(val)
	case []uint64:
		return convertSliceIntegral(val)
	case []uint:
		return convertSliceIntegral(val)
	case []uintptr:
		return convertSliceIntegral(val)
	case []float32:
		return convertSliceFloat(val)
	case []float64:
		return attribute.Float64SliceValue(val)
	case []bool:
		return attribute.BoolSliceValue(val)
	case []string:
		return attribute.StringSliceValue(val)
	case []any:
		if res, ok := convertAnySliceIntegral[int8](val); ok {
			return res
		}

		if res, ok := convertAnySliceIntegral[int16](val); ok {
			return res
		}

		if res, ok := convertAnySliceIntegral[int32](val); ok {
			return res
		}

		if res, ok := convertAnySliceIntegral[int64](val); ok {
			return res
		}

		if res, ok := convertAnySliceIntegral[int](val); ok {
			return res
		}

		if res, ok := convertAnySliceIntegral[uint8](val); ok {
			return res
		}

		if res, ok := convertAnySliceIntegral[uint16](val); ok {
			return res
		}

		if res, ok := convertAnySliceIntegral[uint32](val); ok {
			return res
		}

		if res, ok := convertAnySliceIntegral[uint64](val); ok {
			return res
		}

		if res, ok := convertAnySliceIntegral[uint](val); ok {
			return res
		}

		if res, ok := convertAnySliceIntegral[uintptr](val); ok {
			return res
		}

		if res, ok := convertAnySliceFloat[float32](val); ok {
			return res
		}

		if res, ok := convertAnySliceFloat[float64](val); ok {
			return res
		}

		if res, ok := convertAnySlice[bool](val); ok {
			return attribute.BoolSliceValue(res)
		}

		if res, ok := convertAnySlice[string](val); ok {
			return attribute.StringSliceValue(res)
		}

		res := make([]string, len(val))

		for i, v := range val {
			res[i] = marshalJSON(v)
		}

		return attribute.StringSliceValue(res)
	case json.Marshaler:
		return attribute.StringValue(marshalJSON(val))
	case fmt.Stringer:
		return attribute.StringValue(val.String())
	default:
		valKind := reflect.TypeOf(val).Kind()
		if valKind == reflect.Slice || valKind == reflect.Array {
			reflectVal := reflect.ValueOf(val)
			result := make([]string, reflectVal.Len())

			for i := 0; i < reflectVal.Len(); i++ {
				result[i] = marshalJSON(reflectVal.Index(i).Interface())
			}

			return attribute.StringSliceValue(result)
		} else {
			return attribute.StringValue(marshalJSON(val))
		}
	}
}

func OtelAttr(att Attr) attribute.KeyValue {
	return attribute.KeyValue{Key: attribute.Key(att.Key), Value: AnyToOtelValue(att.Value.Any())}
}

func OtelValue(val Value) attribute.Value {
	return AnyToOtelValue(val.Any())
}
