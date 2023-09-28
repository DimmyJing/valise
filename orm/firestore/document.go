package firestore

import (
	"context"
	"reflect"

	"cloud.google.com/go/firestore"
	"github.com/DimmyJing/valise/attr"
	"github.com/DimmyJing/valise/ctx"
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
	ctx, end := ctx.Nested("transactionData", attr.String("path", d.Ref.Path))
	defer end()

	snap, err := tx.Get(d.Ref)
	if err != nil {
		return *new(D), ctx.Fail(err)
	}

	res, err := d.DataFrom(snap)

	return res, ctx.FailIf(err)
}

func (d *Doc[D]) Trans(cctx ctx.Context, transFn func(D) (D, error)) error {
	cctx, end := cctx.Nested("documentTransaction", attr.String("path", d.Ref.Path))
	defer end()

	err := d.Client.RunTransaction(cctx, func(c context.Context, txn *firestore.Transaction) error {
		ctx, end := ctx.From(c).Nested("documentTransactionInternal")
		defer end()

		data, err := d.transactionData(ctx, txn)
		if err != nil {
			return ctx.Fail(err)
		}

		data, err = func() (D, error) {
			ctx, end := ctx.Nested("documentTransactionInternalFn")
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

		return ctx.FailIf(txn.Set(d.Ref, transformedData))
	})

	return cctx.FailIf(err)
}

func (d *Doc[D]) Set(ctx ctx.Context, data D) (*firestore.WriteResult, error) {
	ctx, end := ctx.Nested("setDocument", attr.String("path", d.Ref.Path))
	defer end()

	transformedData, err := transformStruct(data, false)
	if err != nil {
		return nil, ctx.Fail(err)
	}

	res, err := d.Ref.Set(ctx, transformedData)

	return res, ctx.FailIf(err)
}

func (d *Doc[D]) Delete(ctx ctx.Context) (*firestore.WriteResult, error) {
	ctx, end := ctx.Nested("deleteDocument", attr.String("path", d.Ref.Path))
	defer end()

	res, err := d.Ref.Delete(ctx)

	return res, ctx.FailIf(err)
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

		path, _, exported := getFieldInfo(field)
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
