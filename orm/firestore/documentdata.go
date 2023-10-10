package firestore

import (
	"errors"
	"fmt"
	"reflect"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/DimmyJing/valise/attr"
	"github.com/DimmyJing/valise/ctx"
	"github.com/DimmyJing/valise/jsonschema"
	"github.com/DimmyJing/valise/utils"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	errInvalidStruct    = errors.New("invalid struct")
	errMissingMigration = errors.New("missing migration")
)

func transformStruct(data any, create bool) (map[string]any, error) {
	res, err := jsonschema.ValueToAny(reflect.ValueOf(data))
	if err != nil {
		return nil, fmt.Errorf("failed to convert value %v to any: %w", data, err)
	}

	resMap, ok := res.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("failed to convert message to map %v, got %v - %T: %w", data, res, res, errInvalidStruct)
	}

	if versioner, ok := data.(Versioner); ok {
		resMap["version"] = versioner.CurrentVersion()
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
	data := *new(D)
	if !snap.Exists() {
		return data, ErrDocumentNotFound
	}

	rawData := snap.Data()

	needMigration := false

	var err error

	//nolint:nestif
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

	if !needMigration {
		if err = jsonschema.AnyToValue(rawData, reflect.ValueOf(&data).Elem()); err == nil {
			return data, nil
		}
	} else {
		err = errMissingMigration
	}

	if migrater, ok := any(data).(Migrater); ok {
		rawData = migrater.Migrate(rawData)
		data = *new(D)

		if err := jsonschema.AnyToValue(rawData, reflect.ValueOf(&data).Elem()); err != nil {
			return data, fmt.Errorf("failed to convert data to D after migration: %w", err)
		}

		return data, nil
	}

	return data, fmt.Errorf("failed to fill struct with data: %w", err)
}

var ErrDocumentNotFound = errors.New("document not found")

func (d *Doc[D]) Data(ctx ctx.Context) (D, error) { //nolint:ireturn
	ctx, end := ctx.NestedClient("firestore.data",
		attr.String("path", d.Ref.Path),
		attr.String(string(semconv.DBSystemKey), "firestore"),
		attr.String(string(semconv.DBNameKey), getDBName(d.Ref.Path)),
		attr.String(string(semconv.DBOperationKey), "data"),
	)
	defer end()

	if snap, err := d.Ref.Get(ctx); err != nil {
		if status.Code(err) == codes.NotFound {
			return *new(D), ErrDocumentNotFound
		}

		return *new(D), fmt.Errorf("error getting document snapshot: %w", err)
	} else {
		return d.DataFrom(snap)
	}
}
