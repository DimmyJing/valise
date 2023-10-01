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
	"github.com/DimmyJing/valise/jsonschema"
	"google.golang.org/protobuf/proto"
)

var (
	errInvalidStruct    = errors.New("invalid struct")
	errMissingMigration = errors.New("missing migration")
)

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

func transformStruct(data proto.Message, create bool) (map[string]any, error) {
	res := jsonschema.MessageToAny(data.ProtoReflect())

	resMap, ok := res.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("failed to convert message to map %v, got %v - %T: %w", data, res, res, errInvalidStruct)
	}

	if updatedAt, ok := resMap["updatedAt"]; ok {
		if _, ok := updatedAt.(time.Time); ok {
			resMap["updatedAt"] = time.Now().UTC()
		}
	}

	if createdAt, ok := resMap["createdAt"]; ok && create {
		if _, ok := createdAt.(time.Time); ok {
			resMap["createdAt"] = time.Now().UTC()
		}
	}

	return resMap, nil
}

// TODO: figure out what to do with migrations.
// TODO: figure out what to do with versions.

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

var ErrDataDoesNotExist = errors.New("data does not exist")

func (d *Doc[D]) DataFrom(snap *firestore.DocumentSnapshot) (D, error) { //nolint:ireturn
	data := *new(D)
	if !snap.Exists() {
		return data, ErrDataDoesNotExist
	}

	rawData := snap.Data()

	needMigration := false

	var err error

	/*
		if versioner, ok := any(data).(Versioner); ok {
			if version, ok := rawData["version"]; ok {
				if versionStr, ok := version.(string); ok {
					res, err := utils.CompareSemVer(versioner.CurrentVersion(), versionStr)
					if err != nil {
						return data, fmt.Errorf("failed to compare versions: %w", err)
					}

					needMigration = res > 0
				}
			}
		}
	*/

	//nolint:nestif
	if !needMigration {
		msg := data.ProtoReflect().New()
		if err := jsonschema.AnyToMessage(rawData, msg); err == nil {
			if dat, ok := msg.Interface().(D); ok {
				return dat, nil
			} else {
				return data, fmt.Errorf("failed to convert data to D: %w", err)
			}
		}
	} else {
		err = errMissingMigration
	}

	/*
		if migrater, ok := any(data).(Migrater); ok {
			rawData = migrater.Migrate(rawData)

			msg := data.ProtoReflect().New()

			err := jsonschema.AnyToMessage(rawData, msg)
			if err != nil {
				return data, fmt.Errorf("failed to convert data to D after migration: %w", err)
			}

			if dat, ok := msg.Interface().(D); ok {
				data = dat
			} else {
				return data, fmt.Errorf("failed to convert data to D: %w", err)
			}
		}
	*/

	return data, fmt.Errorf("failed to fill struct with data: %w", err)
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
