package firestore

import (
	"context"
	"reflect"
	"strings"

	"cloud.google.com/go/firestore"
	"github.com/DimmyJing/valise/attr"
	"github.com/DimmyJing/valise/ctx"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

type documentInterface interface {
	setRef(ref *firestore.DocumentRef)
	getRef() *firestore.DocumentRef
	setClient(client *firestore.Client)
	getClient() *firestore.Client
}

type Doc[D any] struct {
	Ref    *firestore.DocumentRef
	Client *firestore.Client
}

func (d *Doc[D]) getRef() *firestore.DocumentRef {
	return d.Ref
}

func (d *Doc[D]) setRef(ref *firestore.DocumentRef) {
	d.Ref = ref
}

func (d *Doc[D]) getClient() *firestore.Client {
	return d.Client
}

func (d *Doc[D]) setClient(client *firestore.Client) {
	d.Client = client
}

func (d *Doc[D]) transactionData(ctx ctx.Context, tx *firestore.Transaction) (D, error) { //nolint:ireturn
	ctx, end := ctx.Nested("firestore.update.transaction.data", attr.String("path", d.Ref.Path))
	defer end()

	snap, err := tx.Get(d.Ref)
	if err != nil {
		return *new(D), ctx.Fail(err)
	}

	res, err := d.DataFrom(snap)

	return res, ctx.FailIf(err)
}

func getDBName(path string) string {
	splitPath := strings.Split(path, "/")

	return splitPath[1] + "/" + splitPath[3]
}

func (d *Doc[D]) Trans(cctx ctx.Context, transFn func(D) (D, error)) error {
	cctx, end := cctx.NestedClient("firestore.update",
		attr.String("path", d.Ref.Path),
		attr.String(string(semconv.DBSystemKey), "firestore"),
		attr.String(string(semconv.DBNameKey), getDBName(d.Ref.Path)),
		attr.String(string(semconv.DBOperationKey), "update"),
	)
	defer end()

	var (
		initData    D
		updatedData map[string]any
	)

	err := d.Client.RunTransaction(cctx, func(c context.Context, txn *firestore.Transaction) error {
		ctx, end := ctx.From(c).Nested("firestore.update.transaction")
		defer end()

		data, err := d.transactionData(ctx, txn)
		if err != nil {
			return ctx.Fail(err)
		}

		initData = data

		data, err = func() (D, error) {
			ctx, end := ctx.Nested("firestore.update.transaction.transform")
			defer end()

			data, err := transFn(data)

			return data, ctx.FailIf(err)
		}()
		if err != nil {
			return ctx.Fail(err)
		}

		transformedData, err := transformStruct(data, false)
		if err != nil {
			return ctx.Fail(err)
		}

		updatedData = transformedData

		return ctx.FailIf(txn.Set(d.Ref, transformedData))
	})

	dumpData(cctx, "update", d.Ref.Path, updatedData, initData)

	return cctx.FailIf(err)
}

func (d *Doc[D]) Set(ctx ctx.Context, data D) (*firestore.WriteResult, error) {
	ctx, end := ctx.NestedClient("firestore.set",
		attr.String("path", d.Ref.Path),
		attr.String(string(semconv.DBSystemKey), "firestore"),
		attr.String(string(semconv.DBNameKey), getDBName(d.Ref.Path)),
		attr.String(string(semconv.DBOperationKey), "set"),
	)
	defer end()

	transformedData, err := transformStruct(data, false)
	if err != nil {
		return nil, ctx.Fail(err)
	}

	res, err := d.Ref.Set(ctx, transformedData)

	dumpData(ctx, "set", d.Ref.Path, transformedData, nil)

	return res, ctx.FailIf(err)
}

func (d *Doc[D]) Delete(ctx ctx.Context) (*firestore.WriteResult, error) {
	ctx, end := ctx.NestedClient("firestore.delete",
		attr.String("path", d.Ref.Path),
		attr.String(string(semconv.DBSystemKey), "firestore"),
		attr.String(string(semconv.DBNameKey), getDBName(d.Ref.Path)),
		attr.String(string(semconv.DBOperationKey), "delete"),
	)
	defer end()

	res, err := d.Ref.Delete(ctx)

	dumpData(ctx, "delete", d.Ref.Path, nil, nil)

	return res, ctx.FailIf(err)
}

func getFieldInfo(fieldType reflect.StructField) (string, bool) {
	fieldName := fieldType.Name
	exported := fieldType.IsExported()

	if tag, ok := fieldType.Tag.Lookup("json"); ok {
		splitTag := strings.Split(tag, ",")
		if len(splitTag) > 0 {
			if splitTag[0] == "-" {
				return "", false
			} else if splitTag[0] != "" {
				fieldName = splitTag[0]
			}
		}
	}

	return fieldName, exported
}

func createDocument[Type any]( //nolint:ireturn
	ref *firestore.DocumentRef,
	client *firestore.Client,
) Type {
	var obj Type
	//nolint:forcetypeassert
	objPtr := any(&obj).(documentInterface)
	objPtr.setRef(ref)
	objPtr.setClient(client)

	objValue := reflect.ValueOf(objPtr).Elem()
	objType := objValue.Type()

	for fieldIdx := 0; fieldIdx < objValue.NumField(); fieldIdx++ {
		field := objType.Field(fieldIdx)

		path, exported := getFieldInfo(field)
		if !exported || field.Type.Kind() != reflect.Struct {
			continue
		}

		if collection, ok := objValue.Field(fieldIdx).Addr().Interface().(collectionInterface); ok {
			collection.setPath(path)
			collection.setClient(objPtr.getClient())
			collection.setParent(objPtr.getRef())
		}
	}

	return obj
}

func CreateRoot[Doc any](client *firestore.Client) Doc { //nolint:ireturn
	return createDocument[Doc](nil, client)
}
