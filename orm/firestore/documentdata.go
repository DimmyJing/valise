package firestore

import (
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/DimmyJing/valise/attr"
	"github.com/DimmyJing/valise/ctx"
	"github.com/DimmyJing/valise/utils"
)

var (
	errDataNotStruct    = errors.New("data is not a struct")
	errInvalidStruct    = errors.New("invalid struct")
	errMissingField     = errors.New("missing field")
	errExtraFields      = errors.New("extra fields")
	errMissingMigration = errors.New("missing migration")
)

func fillValue(value reflect.Value, rawData any) error { //nolint:funlen,gocognit,gocyclo,cyclop
	//nolint:nestif
	if value.Kind() == reflect.Ptr {
		if rawData == nil {
			value.Set(reflect.Zero(value.Type()))

			return nil
		}

		if value.IsNil() {
			value.Set(reflect.New(value.Type().Elem()))
		}

		err := fillValue(value.Elem(), rawData)
		if err != nil {
			return fmt.Errorf("cannot set pointer on %v: %w", value, errInvalidStruct)
		}

		return nil
	} else if value.Kind() == reflect.Interface {
		if value.NumMethod() != 0 {
			return fmt.Errorf("cannot set %v on %v: %w", rawData, value, errInvalidStruct)
		}

		if rawData == nil {
			value.SetZero()

			return nil
		}

		value.Set(reflect.ValueOf(rawData))

		return nil
	}

	switch data := rawData.(type) {
	case nil:
		//nolint:exhaustive
		switch value.Kind() {
		case reflect.Map:
			value.Set(reflect.MakeMap(value.Type()))
		case reflect.Slice:
			value.Set(reflect.MakeSlice(value.Type(), 0, 0))
		default:
			return fmt.Errorf("cannot set nil on %v: %w", value, errInvalidStruct)
		}
	case []byte:
		if value.Kind() == reflect.Slice && value.Type().Elem().Kind() == reflect.Uint8 {
			value.SetBytes(data)
		} else {
			return fmt.Errorf("cannot set []byte on %v: %w", value, errInvalidStruct)
		}
	case time.Time:
		dataVal := reflect.ValueOf(data)
		if dataVal.Type().AssignableTo(value.Type()) {
			value.Set(reflect.ValueOf(data))
		} else {
			return fmt.Errorf("cannot set time.Time on %v: %w", value, errInvalidStruct)
		}
	case *firestore.DocumentRef:
		dataVal := reflect.ValueOf(data).Elem()
		if dataVal.Type().AssignableTo(value.Type()) {
			value.Set(dataVal)
		} else {
			return fmt.Errorf("cannot set *firestore.DocumentRef on %v: %w", value, errInvalidStruct)
		}
	case bool:
		if value.Kind() == reflect.Bool {
			value.SetBool(data)
		} else {
			return fmt.Errorf("cannot set bool on %v: %w", value, errInvalidStruct)
		}
	case string:
		if value.Kind() == reflect.String {
			value.SetString(data)
		} else {
			return fmt.Errorf("cannot set string on %v: %w", value, errInvalidStruct)
		}
	case int64:
		switch {
		case value.CanInt():
			value.SetInt(data)
		case value.CanUint():
			value.SetUint(uint64(data))
		default:
			return fmt.Errorf("cannot set int64 on %v: %w", value, errInvalidStruct)
		}
	case float64:
		if value.CanFloat() {
			value.SetFloat(data)
		} else {
			return fmt.Errorf("cannot set float64 on %v: %w", value, errInvalidStruct)
		}
	case []any:
		switch {
		case value.Kind() == reflect.Slice:
			value.Set(reflect.MakeSlice(value.Type(), len(data), len(data)))

			for idx, item := range data {
				err := fillValue(value.Index(idx), item)
				if err != nil {
					return fmt.Errorf("cannot set item %d on %v: %w", idx, value, err)
				}
			}
		case value.Kind() == reflect.Array:
			vLen := value.Len()
			if vLen > len(data) {
				z := reflect.Zero(value.Type().Elem())
				for i := len(data); i < vLen; i++ {
					value.Index(i).Set(z)
				}
			}

			minLen := min(vLen, len(data))
			for idx := 0; idx < minLen; idx++ {
				err := fillValue(value.Index(idx), data[idx])
				if err != nil {
					return fmt.Errorf("cannot set item %d on %v: %w", idx, value, err)
				}
			}
		default:
			return fmt.Errorf("cannot set []any on %v: %w", value, errInvalidStruct)
		}
	case map[string]any:
		switch {
		case value.Kind() == reflect.Struct:
			err := fillStruct(value, data)
			if err != nil {
				return fmt.Errorf("cannot set map[string]any on %v: %w", value, err)
			}
		case value.Kind() == reflect.Map:
			if value.IsNil() {
				value.Set(reflect.MakeMap(value.Type()))
			}

			for key, item := range data {
				zero := reflect.New(value.Type().Elem()).Elem()

				err := fillValue(zero, item)
				if err != nil {
					return fmt.Errorf("cannot set item %s on %v: %w", key, value, err)
				}

				value.SetMapIndex(reflect.ValueOf(key), zero)
			}
		default:
			return fmt.Errorf("cannot set map[string]any on %v: %w", value, errInvalidStruct)
		}
	default:
		return fmt.Errorf("%v is not supported: %w", data, errInvalidStruct)
	}

	return nil
}

func getFieldInfo(fieldType reflect.StructField) (string, bool, bool) {
	fieldName := fieldType.Name
	required := true
	exported := fieldType.IsExported()

	//nolint:nestif
	if tag, ok := fieldType.Tag.Lookup("json"); ok {
		splitTag := strings.Split(tag, ",")
		if len(splitTag) > 0 {
			if splitTag[0] == "-" {
				return "", false, false
			} else if splitTag[0] != "" {
				fieldName = splitTag[0]
			}

			if slices.Contains(splitTag, "omitempty") {
				required = false
			}
		}
	}

	return fieldName, required, exported
}

func fillStruct(data reflect.Value, rawData map[string]any) error {
	elemType := data.Type()

	processedKeys := make(map[string]bool)

	for fieldIdx := 0; fieldIdx < elemType.NumField(); fieldIdx++ {
		fieldName, required, exported := getFieldInfo(elemType.Field(fieldIdx))
		if !exported {
			continue
		}

		if rawField, ok := rawData[fieldName]; ok {
			if err := fillValue(data.Field(fieldIdx), rawField); err != nil {
				return fmt.Errorf("failed to fill field %s: %w", fieldName, err)
			}
		} else if required {
			return fmt.Errorf("required field %s is missing: %w", fieldName, errMissingField)
		}

		processedKeys[fieldName] = true
	}

	if len(processedKeys) < len(rawData) {
		res := []string{}

		for key := range rawData {
			if _, ok := processedKeys[key]; !ok {
				res = append(res, key)
			}
		}

		return fmt.Errorf("extra fields %v: %w", res, errExtraFields)
	}

	return nil
}

func fillFromValue(value reflect.Value) (any, error) { //nolint:funlen,gocognit,cyclop
	if value.Kind() == reflect.Ptr {
		if value.IsNil() {
			//nolint:nilnil
			return nil, nil
		}

		val, err := fillFromValue(value.Elem())
		if err != nil {
			return nil, fmt.Errorf("cannot fill from pointer %v: %w", value, errInvalidStruct)
		}

		return val, nil
	}

	switch val := value.Interface().(type) {
	case []byte:
		return val, nil
	case time.Time:
		return val, nil
	case firestore.DocumentRef:
		return &val, nil
	}

	//nolint:exhaustive
	switch value.Kind() {
	case reflect.Bool:
		return value.Bool(), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return value.Int(), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return int64(value.Uint()), nil
	case reflect.Float32, reflect.Float64:
		return value.Float(), nil
	case reflect.String:
		return value.String(), nil
	case reflect.Array:
		res := make([]any, value.Len())

		for idx := 0; idx < value.Len(); idx++ {
			val, err := fillFromValue(value.Index(idx))
			if err != nil {
				return nil, fmt.Errorf("cannot fill from array at %d, %v: %w", idx, value, err)
			}

			res[idx] = val
		}

		return res, nil
	case reflect.Slice:
		res := make([]any, value.Len())

		if value.IsNil() {
			return res, nil
		}

		for idx := 0; idx < value.Len(); idx++ {
			val, err := fillFromValue(value.Index(idx))
			if err != nil {
				return nil, fmt.Errorf("cannot fill from slice at %d, %v: %w", idx, value, err)
			}

			res[idx] = val
		}

		return res, nil
	case reflect.Map:
		if value.Type().Key().Kind() != reflect.String {
			return nil, fmt.Errorf("cannot fill from map with non-string key %v: %w", value, errInvalidStruct)
		} else if value.IsNil() {
			return make(map[string]any), nil
		}

		res := make(map[string]any)

		for _, key := range value.MapKeys() {
			if val, err := fillFromValue(value.MapIndex(key)); err == nil {
				res[key.String()] = val
			} else {
				return nil, fmt.Errorf("cannot fill from map at %s, %v: %w", key.String(), value, err)
			}
		}

		return res, nil
	case reflect.Struct:
		if val, err := fillFromStruct(value); err == nil {
			return val, nil
		} else {
			return nil, fmt.Errorf("cannot fill from struct %v: %w", value, err)
		}
	case reflect.Interface:
		if value.NumMethod() == 0 {
			return fillFromValue(value.Elem())
		} else {
			return nil, fmt.Errorf("cannot fill from nonempty interface %v: %w", value, errInvalidStruct)
		}
	default:
		return nil, fmt.Errorf("cannot fill from value %v: %w", value, errInvalidStruct)
	}
}

func fillFromStruct(data reflect.Value) (map[string]any, error) {
	rawData := make(map[string]any)
	elemType := data.Type()

	for fieldIdx := 0; fieldIdx < elemType.NumField(); fieldIdx++ {
		fieldName, required, exported := getFieldInfo(elemType.Field(fieldIdx))
		if !exported {
			continue
		}

		var err error

		fieldData := data.Field(fieldIdx)
		if fieldData.IsZero() && !required {
			continue
		}

		rawData[fieldName], err = fillFromValue(fieldData)
		if err != nil {
			return nil, fmt.Errorf("failed to fill from field %s: %w", fieldName, err)
		}
	}

	return rawData, nil
}

func transformStruct(data any, create bool) (map[string]any, error) {
	dataVal := reflect.ValueOf(data)
	if dataVal.Kind() != reflect.Struct {
		return nil, errDataNotStruct
	}

	res, err := fillFromStruct(reflect.ValueOf(data))
	if err != nil {
		return nil, err
	}

	if versioner, ok := data.(Versioner); ok {
		res["version"] = versioner.CurrentVersion()
	}

	if updatedAt, ok := res["updatedAt"]; ok {
		if _, ok := updatedAt.(time.Time); ok {
			res["updatedAt"] = time.Now().UTC()
		}
	}

	if createdAt, ok := res["createdAt"]; ok && create {
		if _, ok := createdAt.(time.Time); ok {
			res["createdAt"] = time.Now().UTC()
		}
	}

	return res, nil
}

type Migrater interface {
	Migrate(map[string]any) map[string]any
}

type Versioner interface {
	CurrentVersion() string
}

func callDataFrom[Doc any, Data any](doc *Doc, snap *firestore.DocumentSnapshot) (Data, error) { //nolint:ireturn
	docVal := reflect.ValueOf(doc)
	dataFromMethod := docVal.MethodByName("DataFrom")
	res := dataFromMethod.Call([]reflect.Value{reflect.ValueOf(snap)})
	//nolint:forcetypeassert
	if !res[1].IsNil() {
		return *new(Data), fmt.Errorf("error calling DataFrom: %w", res[1].Interface().(error))
	}
	//nolint:forcetypeassert
	return res[0].Interface().(Data), nil
}

func (d *Doc[D]) DataFrom(snap *firestore.DocumentSnapshot) (D, error) { //nolint:ireturn
	data := new(D)

	dataVal := reflect.ValueOf(data).Elem()
	if dataVal.Kind() != reflect.Struct {
		return *data, fmt.Errorf("data is not a struct: %w", errDataNotStruct)
	}

	rawData := snap.Data()
	needMigration := false

	defer func() {
		idField := dataVal.FieldByName("ID")
		if idField.Kind() == reflect.String {
			idField.SetString(snap.Ref.ID)
		}
	}()

	var err error

	if versioner, ok := any(data).(Versioner); ok {
		if _, ok := rawData["version"]; ok {
			res, err := utils.CompareSemVer(versioner.CurrentVersion(), rawData["version"].(string))
			if err != nil {
				return *data, fmt.Errorf("failed to compare versions: %w", err)
			}

			needMigration = res > 0
		}
	}

	if !needMigration {
		err = fillStruct(dataVal, rawData)
		if err == nil {
			return *data, nil
		}
	} else {
		err = errMissingMigration
	}

	if migrater, ok := any(data).(Migrater); ok {
		rawData = migrater.Migrate(rawData)
		data = new(D)
		dataVal = reflect.ValueOf(data).Elem()
		err := fillStruct(dataVal, rawData)

		if err != nil {
			return *data, fmt.Errorf("failed to fill struct with data after migration: %w", err)
		} else {
			return *data, nil
		}
	}

	return *data, fmt.Errorf("failed to fill struct with data: %w", err)
}

func (d *Doc[D]) Data(ctx ctx.Context) (D, error) { //nolint:ireturn
	ctx, end := ctx.Nested("getDocument", attr.String("path", d.Ref.Path))
	defer end()

	if snap, err := d.Ref.Get(ctx); err != nil {
		return *new(D), fmt.Errorf("error getting document snapshot: %w", err)
	} else {
		return d.DataFrom(snap)
	}
}
