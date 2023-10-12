package jsonschema

import (
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"
)

func ValueToAny(value reflect.Value) (any, error) { //nolint:cyclop,funlen,gocognit,gocyclo
	//nolint:exhaustive
	switch value.Type().Kind() {
	case reflect.Bool:
		return value.Bool(), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return value.Int(), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return value.Uint(), nil
	case reflect.Float32, reflect.Float64:
		return value.Float(), nil
	case reflect.Array:
		result := make([]any, value.Len())

		for idx := 0; idx < value.Len(); idx++ {
			arrayVal, err := ValueToAny(value.Index(idx))
			if err != nil {
				return nil, fmt.Errorf("failed to convert array value at idx %d: %w", idx, err)
			}

			result[idx] = arrayVal
		}

		return result, nil
	case reflect.Interface:
		if value.NumMethod() != 0 {
			return nil, fmt.Errorf("invalid interface type: %w", errReflectType)
		}

		if value.IsNil() {
			return value.Interface(), nil
		}

		return ValueToAny(value.Elem())
	case reflect.Map:
		if value.Type().Key().Kind() != reflect.String {
			return nil, fmt.Errorf("invalid map key type %s: %w", value.Type().Key().Kind().String(), errReflectType)
		}

		result := make(map[string]any)

		iter := value.MapRange()
		for iter.Next() {
			mapValue, err := ValueToAny(iter.Value())
			if err != nil {
				return nil, fmt.Errorf("failed to convert map value at key %s: %w", iter.Key().String(), err)
			}

			result[iter.Key().String()] = mapValue
		}

		return result, nil
	case reflect.Pointer:
		if value.IsNil() {
			//nolint:nilnil
			return nil, nil
		}

		ptrValue, err := ValueToAny(value.Elem())
		if err != nil {
			return nil, fmt.Errorf("failed to convert pointer value: %w", err)
		}

		return ptrValue, nil
	case reflect.Slice:
		if value.Type().Elem().Kind() == reflect.Uint8 {
			return value.Bytes(), nil
		}

		result := make([]any, value.Len())

		for idx := 0; idx < value.Len(); idx++ {
			sliceVal, err := ValueToAny(value.Index(idx))
			if err != nil {
				return nil, fmt.Errorf("failed to convert slice value at idx %d: %w", idx, err)
			}

			result[idx] = sliceVal
		}

		return result, nil
	case reflect.String:
		return value.String(), nil
	case reflect.Struct:
		if timeValue, isTime := value.Interface().(time.Time); isTime {
			return timeValue, nil
		}

		result := make(map[string]any)

		for idx := 0; idx < value.NumField(); idx++ {
			field := value.Type().Field(idx)
			if !field.IsExported() {
				continue
			}

			fieldName := strings.ToLower(string(field.Name[0])) + field.Name[1:]
			optional := false

			//nolint:nestif
			if jsonTag, found := field.Tag.Lookup("json"); found {
				splitTags := strings.Split(jsonTag, ",")
				if len(splitTags) > 0 {
					if splitTags[0] == "-" && len(splitTags) == 1 {
						continue
					} else if splitTags[0] != "" {
						fieldName = splitTags[0]
					}
				}

				if slices.Contains(splitTags[1:], "omitempty") {
					optional = true
				}
			}

			fieldValue, err := ValueToAny(value.Field(idx))
			if err != nil {
				return nil, fmt.Errorf("failed to convert struct field %s: %w", fieldName, err)
			}

			if optional && (fieldValue == nil || reflect.ValueOf(fieldValue).IsZero()) {
				continue
			}

			result[fieldName] = fieldValue
		}

		return result, nil
	default:
		return nil, fmt.Errorf("invalid reflect type %s: %w", value.Kind().String(), errReflectType)
	}
}

func AnyToValue(anyVal any, value reflect.Value) error { //nolint:funlen,gocognit,gocyclo,cyclop,maintidx
	if !value.CanSet() {
		return fmt.Errorf("value is not settable: %w", errReflectType)
	}
	//nolint:exhaustive
	switch value.Type().Kind() {
	case reflect.Bool:
		//nolint:nestif
		if boolVal, isBool := anyVal.(bool); isBool {
			value.SetBool(boolVal)
		} else if stringVal, isString := anyVal.(string); isString {
			res, err := strconv.ParseBool(stringVal)
			if err != nil {
				return fmt.Errorf("invalid bool string value %s: %w", stringVal, errReflectType)
			}

			value.SetBool(res)
		} else if stringSliceVal, isStringSlice := anyVal.([]string); isStringSlice {
			if len(stringSliceVal) != 1 {
				return fmt.Errorf("invalid string slice length %d: %w", len(stringSliceVal), errReflectType)
			} else if res, err := strconv.ParseBool(stringSliceVal[0]); err != nil {
				return fmt.Errorf("invalid bool string value %s: %w", stringSliceVal[0], errReflectType)
			} else {
				value.SetBool(res)
			}
		} else {
			return fmt.Errorf("invalid bool value %v: %w", anyVal, errReflectType)
		}
	// TODO: make this faster by not using reflect
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		anyElem := reflect.ValueOf(anyVal)
		//nolint:nestif
		if anyElem.CanConvert(value.Type()) {
			value.Set(anyElem.Convert(value.Type()))
		} else if stringVal, isString := anyVal.(string); isString {
			intVal, err := strconv.ParseInt(stringVal, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid int string value %s: %w", stringVal, errReflectType)
			}

			value.SetInt(intVal)
		} else if stringSliceVal, isStringSlice := anyVal.([]string); isStringSlice {
			if len(stringSliceVal) != 1 {
				return fmt.Errorf("invalid string slice length %d: %w", len(stringSliceVal), errReflectType)
			} else if res, err := strconv.ParseInt(stringSliceVal[0], 10, 64); err != nil {
				return fmt.Errorf("invalid int string value %s: %w", stringSliceVal[0], errReflectType)
			} else {
				value.SetInt(res)
			}
		} else {
			return fmt.Errorf("cannot set int value from %v: %w", anyVal, errReflectType)
		}
	// TODO: make this faster by not using reflect
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		anyElem := reflect.ValueOf(anyVal)
		//nolint:nestif
		if anyElem.CanConvert(value.Type()) {
			value.Set(anyElem.Convert(value.Type()))
		} else if stringVal, isString := anyVal.(string); isString {
			intVal, err := strconv.ParseUint(stringVal, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid int string value %s: %w", stringVal, errReflectType)
			}

			value.SetUint(intVal)
		} else if stringSliceVal, isStringSlice := anyVal.([]string); isStringSlice {
			if len(stringSliceVal) != 1 {
				return fmt.Errorf("invalid string slice length %d: %w", len(stringSliceVal), errReflectType)
			} else if res, err := strconv.ParseUint(stringSliceVal[0], 10, 64); err != nil {
				return fmt.Errorf("invalid uint string value %s: %w", stringSliceVal[0], errReflectType)
			} else {
				value.SetUint(res)
			}
		} else {
			return fmt.Errorf("cannot set uint value from %v: %w", anyVal, errReflectType)
		}
	// TODO: make this faster by not using reflect
	case reflect.Float32, reflect.Float64:
		anyElem := reflect.ValueOf(anyVal)
		//nolint:nestif
		if anyElem.CanConvert(value.Type()) {
			value.Set(anyElem.Convert(value.Type()))
		} else if stringVal, isString := anyVal.(string); isString {
			floatVal, err := strconv.ParseFloat(stringVal, 64)
			if err != nil {
				return fmt.Errorf("invalid int string value %s: %w", stringVal, errReflectType)
			}

			value.SetFloat(floatVal)
		} else if stringSliceVal, isStringSlice := anyVal.([]string); isStringSlice {
			if len(stringSliceVal) != 1 {
				return fmt.Errorf("invalid string slice length %d: %w", len(stringSliceVal), errReflectType)
			} else if res, err := strconv.ParseFloat(stringSliceVal[0], 64); err != nil {
				return fmt.Errorf("invalid float string value %s: %w", stringSliceVal[0], errReflectType)
			} else {
				value.SetFloat(res)
			}
		} else {
			return fmt.Errorf("cannot set float value from %v: %w", anyVal, errReflectType)
		}
	case reflect.Array:
		//nolint:nestif
		if arrayVal, isArray := anyVal.([]any); isArray {
			if len(arrayVal) != value.Len() {
				return fmt.Errorf("invalid array length %d: %w", len(arrayVal), errReflectType)
			}

			for idx, arrayElem := range arrayVal {
				if err := AnyToValue(arrayElem, value.Index(idx)); err != nil {
					return fmt.Errorf("failed to set array value at idx %d: %w", idx, err)
				}
			}
		} else if strArrayVal, isArray := anyVal.([]string); isArray {
			if len(strArrayVal) != value.Len() {
				return fmt.Errorf("invalid array length %d: %w", len(strArrayVal), errReflectType)
			}

			for idx, arrayElem := range strArrayVal {
				if err := AnyToValue(arrayElem, value.Index(idx)); err != nil {
					return fmt.Errorf("failed to set str array value at idx %d: %w", idx, err)
				}
			}
		} else {
			return fmt.Errorf("cannot set array value from %v: %w", anyVal, errReflectType)
		}
	case reflect.Interface:
		if value.NumMethod() != 0 {
			return fmt.Errorf("invalid interface type: %w", errReflectType)
		}

		if anyVal == nil {
			value.SetZero()
		} else {
			value.Set(reflect.ValueOf(anyVal))
		}
	case reflect.Map:
		if value.Type().Key().Kind() != reflect.String {
			return fmt.Errorf("invalid map key type %s: %w", value.Type().Key().Kind().String(), errReflectType)
		}

		value.Set(reflect.MakeMap(value.Type()))

		if mapVal, isMapVal := anyVal.(map[string]any); isMapVal {
			value.Set(reflect.MakeMap(value.Type()))

			for key, mapElem := range mapVal {
				mapValue := reflect.New(value.Type().Elem()).Elem()
				if err := AnyToValue(mapElem, mapValue); err != nil {
					return fmt.Errorf("failed to set map value at key %s: %w", key, err)
				}

				value.SetMapIndex(reflect.ValueOf(key), mapValue)
			}
		} else if anyVal != nil {
			return fmt.Errorf("invalid map value %v: %w", anyVal, errReflectType)
		}
	case reflect.Ptr:
		if anyVal == nil {
			value.SetZero()
		} else {
			if value.IsNil() {
				value.Set(reflect.New(value.Type().Elem()))
			}

			err := AnyToValue(anyVal, value.Elem())
			if err != nil {
				return fmt.Errorf("failed to convert pointer value %v: %w", anyVal, err)
			}
		}
	case reflect.Slice:
		switch {
		case value.Type().Elem().Kind() == reflect.Uint8:
			if byteVal, isByteVal := anyVal.([]byte); isByteVal {
				value.SetBytes(byteVal)
			} else {
				return fmt.Errorf("invalid byte value %v: %w", anyVal, errReflectType)
			}
		case anyVal == nil:
			value.Set(reflect.MakeSlice(value.Type(), 0, 0))
		default:
			if sliceVal, isSliceVal := anyVal.([]any); isSliceVal {
				value.Set(reflect.MakeSlice(value.Type(), len(sliceVal), len(sliceVal)))

				for idx, sliceElem := range sliceVal {
					if err := AnyToValue(sliceElem, value.Index(idx)); err != nil {
						return fmt.Errorf("failed to set slice value at idx %d: %w", idx, err)
					}
				}
			} else if sliceStrVal, isSliceStrVal := anyVal.([]string); isSliceStrVal {
				value.Set(reflect.MakeSlice(value.Type(), len(sliceStrVal), len(sliceStrVal)))

				for idx, sliceElem := range sliceStrVal {
					if err := AnyToValue(sliceElem, value.Index(idx)); err != nil {
						return fmt.Errorf("failed to set str slice value at idx %d: %w", idx, err)
					}
				}
			} else {
				return fmt.Errorf("invalid slice value %v: %w", anyVal, errReflectType)
			}
		}
	case reflect.String:
		//nolint:nestif
		if stringVal, isString := anyVal.(string); isString {
			if enumMember, isEnumMember := value.Interface().(EnumMember); isEnumMember {
				if slices.Contains(enumMember.Members(), stringVal) {
					value.SetString(stringVal)
				} else {
					return fmt.Errorf("invalid enum value %v: %w", anyVal, errReflectType)
				}
			}

			value.SetString(stringVal)
		} else if stringSliceVal, isStringSlice := anyVal.([]string); isStringSlice {
			if len(stringSliceVal) != 1 {
				return fmt.Errorf("invalid string slice length %d: %w", len(stringSliceVal), errReflectType)
			} else {
				value.SetString(stringSliceVal[0])
			}
		} else {
			return fmt.Errorf("invalid string value %v: %w", anyVal, errReflectType)
		}
	case reflect.Struct:
		//nolint:nestif
		if _, isTime := value.Interface().(time.Time); isTime {
			if timeVal, isTimeVal := anyVal.(time.Time); isTimeVal {
				value.Set(reflect.ValueOf(timeVal))
			} else if stringVal, isStringVal := anyVal.(string); isStringVal {
				if timeVal, err := time.Parse(time.RFC3339, stringVal); err != nil {
					return fmt.Errorf("invalid time string value %s: %w", stringVal, errReflectType)
				} else {
					value.Set(reflect.ValueOf(timeVal))
				}
			} else {
				return fmt.Errorf("invalid time value %v: %w", anyVal, errReflectType)
			}
		} else if mapVal, isMapVal := anyVal.(map[string]any); isMapVal {
			processedKeys := make(map[string]struct{})

			for idx := 0; idx < value.NumField(); idx++ {
				field := value.Type().Field(idx)
				if !field.IsExported() {
					continue
				}

				fieldName := strings.ToLower(string(field.Name[0])) + field.Name[1:]
				optional := false

				if jsonTag, found := field.Tag.Lookup("json"); found {
					splitTags := strings.Split(jsonTag, ",")
					if len(splitTags) > 0 {
						if splitTags[0] == "-" && len(splitTags) == 1 {
							continue
						} else if splitTags[0] != "" {
							fieldName = splitTags[0]
						}
					}

					if slices.Contains(splitTags[1:], "omitempty") {
						optional = true
					}
				}

				if fieldVal, found := mapVal[fieldName]; found {
					if err := AnyToValue(fieldVal, value.Field(idx)); err != nil {
						return fmt.Errorf("failed to set struct field %s: %w", fieldName, err)
					}

					processedKeys[fieldName] = struct{}{}
				} else if !optional {
					return fmt.Errorf("missing required field %s: %w", fieldName, errReflectType)
				} else {
					continue
				}
			}

			if len(processedKeys) < len(mapVal) {
				for key := range mapVal {
					if _, found := processedKeys[key]; !found {
						return fmt.Errorf("extra field in struct conversion %s: %w", key, errReflectType)
					}
				}
			}
		} else {
			return fmt.Errorf("invalid struct value %v: %w", anyVal, errReflectType)
		}
	default:
		return fmt.Errorf("invalid reflect type %s: %w", value.Kind().String(), errReflectType)
	}

	return nil
}
