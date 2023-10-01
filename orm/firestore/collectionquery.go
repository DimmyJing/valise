package firestore

import (
	"fmt"
	"reflect"

	"cloud.google.com/go/firestore"
	"github.com/DimmyJing/valise/attr"
	"github.com/DimmyJing/valise/ctx"
	"google.golang.org/api/iterator"
	"google.golang.org/protobuf/proto"
)

type Query = firestore.Query

type docRefIterator[Doc any] struct {
	iter   *firestore.DocumentRefIterator
	client *firestore.Client

	docCache *firestore.DocumentRef
	errCache error
}

func (i *docRefIterator[Doc]) Next() (Doc, error) { //nolint:ireturn
	//nolint:nestif
	if i.docCache != nil || i.errCache != nil {
		defer func() {
			i.docCache = nil
			i.errCache = nil
		}()

		if doc, err := i.docCache, i.errCache; err != nil {
			return *new(Doc), fmt.Errorf("failed getting next document: %w", err)
		} else {
			return createDocument[Doc](doc, i.client), nil
		}
	} else {
		if doc, err := i.iter.Next(); err != nil {
			return *new(Doc), fmt.Errorf("failed getting next document: %w", err)
		} else {
			return createDocument[Doc](doc, i.client), nil
		}
	}
}

func (i *docRefIterator[Doc]) HasNext() bool {
	if i.docCache != nil {
		//nolint:errorlint,goerr113
		return i.errCache != iterator.Done
	}

	i.docCache, i.errCache = i.iter.Next()
	//nolint:errorlint,goerr113
	return i.errCache != iterator.Done
}

type DocSnap[Doc any, D proto.Message] struct {
	Doc  Doc
	Data D
}

type docIterator[Doc any, D proto.Message] struct {
	iter   *firestore.DocumentIterator
	client *firestore.Client

	docCache *firestore.DocumentSnapshot
	errCache error
}

func (i *docIterator[Doc, D]) Next() (DocSnap[Doc, D], error) {
	var snap *firestore.DocumentSnapshot

	//nolint:nestif
	if i.docCache != nil || i.errCache != nil {
		defer func() {
			i.docCache = nil
			i.errCache = nil
		}()

		if doc, err := i.docCache, i.errCache; err != nil {
			return DocSnap[Doc, D]{}, fmt.Errorf("failed getting next document: %w", err)
		} else {
			snap = doc
		}
	} else {
		if doc, err := i.iter.Next(); err != nil {
			return DocSnap[Doc, D]{}, fmt.Errorf("failed getting next document: %w", err)
		} else {
			snap = doc
		}
	}

	doc := createDocument[Doc](snap.Ref, i.client)

	data, err := callDataFrom[Doc, D](&doc, snap)
	if err != nil {
		return DocSnap[Doc, D]{}, fmt.Errorf("failed getting data from snapshot: %w", err)
	}

	return DocSnap[Doc, D]{Doc: doc, Data: data}, nil
}

func (i *docIterator[Doc, D]) HasNext() bool {
	if i.docCache != nil {
		//nolint:errorlint,goerr113
		return i.errCache != iterator.Done
	}

	i.docCache, i.errCache = i.iter.Next()
	//nolint:errorlint,goerr113
	return i.errCache != iterator.Done
}

func (c *Collection[Doc, D]) DocumentsIter(ctx ctx.Context) *docRefIterator[Doc] {
	return &docRefIterator[Doc]{
		iter:   c.Ref().DocumentRefs(ctx),
		client: c.client,

		docCache: nil,
		errCache: nil,
	}
}

func (c *Collection[Doc, D]) Documents(ctx ctx.Context) ([]Doc, error) {
	ctx, end := ctx.Nested("listDocuments", attr.String("path", c.Ref().Path))
	defer end()

	res := make([]Doc, 0)

	it := c.DocumentsIter(ctx)
	for it.HasNext() {
		doc, err := it.Next()
		if err != nil {
			return nil, ctx.Fail(err)
		}

		res = append(res, doc)
	}

	ctx.SetAttributes(attr.Int("count", len(res)))

	return res, nil
}

func (c *Collection[Doc, D]) QueryIter(ctx ctx.Context, queryFns ...func(query Query) Query) *docIterator[Doc, D] {
	query := c.Ref().Query
	for _, queryFn := range queryFns {
		query = queryFn(query)
	}

	return &docIterator[Doc, D]{
		iter:   query.Documents(ctx),
		client: c.client,

		docCache: nil,
		errCache: nil,
	}
}

func (c *Collection[Doc, D]) Query(ctx ctx.Context, queryFns ...func(query Query) Query) ([]DocSnap[Doc, D], error) {
	ctx, end := ctx.Nested("queryDocuments", attr.String("path", c.Ref().Path))
	defer end()

	res := make([]DocSnap[Doc, D], 0)

	it := c.QueryIter(ctx, queryFns...)
	for it.HasNext() {
		doc, err := it.Next()
		if err != nil {
			return nil, ctx.Fail(err)
		}

		res = append(res, doc)
	}

	ctx.SetAttributes(attr.Int("count", len(res)))

	return res, nil
}

func (c *Collection[Doc, D]) GetAll(ctx ctx.Context, documents []Doc) ([]DocSnap[Doc, D], error) {
	ctx, end := ctx.Nested("getAll", attr.String("path", c.Ref().Path))
	defer end()

	docRefs := make([]*firestore.DocumentRef, len(documents))
	for i, doc := range documents {
		//nolint:forcetypeassert
		docRefs[i] = reflect.ValueOf(doc).FieldByName("Ref").Interface().(*firestore.DocumentRef)
	}

	res := make([]DocSnap[Doc, D], 0)

	snaps, err := c.client.GetAll(ctx, docRefs)
	if err != nil {
		return nil, ctx.Fail(err)
	}

	for idx, snap := range snaps {
		data, err := callDataFrom[Doc, D](&documents[idx], snap)
		if err != nil {
			return nil, ctx.Fail(err)
		}

		res[idx] = DocSnap[Doc, D]{
			Doc:  documents[idx],
			Data: data,
		}
	}

	return res, nil
}
