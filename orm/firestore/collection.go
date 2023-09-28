package firestore

import (
	"cloud.google.com/go/firestore"
	"github.com/DimmyJing/valise/attr"
	"github.com/DimmyJing/valise/ctx"
)

type collectionInterface interface {
	setParent(ref *firestore.DocumentRef)
	setPath(path string)
	setClient(client *firestore.Client)
}

type Collection[Doc any, D any] struct {
	client    *firestore.Client
	path      string
	parentRef *firestore.DocumentRef
}

func (c *Collection[Doc, D]) Ref() *firestore.CollectionRef {
	if c.parentRef == nil {
		return c.client.Collection(c.path)
	} else {
		return c.parentRef.Collection(c.path)
	}
}

func (c *Collection[Doc, D]) ID(id string) *Doc {
	doc := createDocument[Doc](c.Ref().Doc(id), c.client)

	return &doc
}

func (c *Collection[Doc, D]) Add(ctx ctx.Context, doc D) (*firestore.DocumentRef, *firestore.WriteResult, error) {
	ctx, end := ctx.Nested("addDocument", attr.String("path", c.Ref().Path))
	defer end()

	transformedData, err := transformStruct(doc, true)
	if err != nil {
		return nil, nil, ctx.Fail(err)
	}

	docRef, writeResult, err := c.Ref().Add(ctx, transformedData)
	if err != nil {
		return nil, nil, ctx.Fail(err)
	}

	ctx.SetAttributes(attr.String("docID", docRef.ID))

	return docRef, writeResult, nil
}

func (c *Collection[Doc, D]) setPath(path string) {
	c.path = path
}

func (c *Collection[Doc, D]) setClient(client *firestore.Client) {
	c.client = client
}

func (c *Collection[Doc, D]) setParent(ref *firestore.DocumentRef) {
	c.parentRef = ref
}
