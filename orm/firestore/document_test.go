package firestore_test

//nolint:dupword
/*
import (
	"bytes"
	"errors"
	"fmt"
	"testing"
	"time"

	origfirestore "cloud.google.com/go/firestore"
	pb "cloud.google.com/go/firestore/apiv1/firestorepb"
	"github.com/DimmyJing/valise/ctx"
	"github.com/DimmyJing/valise/log"
	"github.com/DimmyJing/valise/orm/firestore"
	"github.com/DimmyJing/valise/orm/firestore/mockfs"
	"github.com/stretchr/testify/assert"
	"google.golang.org/genproto/googleapis/type/latlng"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const defVersion = "1.0.0"

type docBuilder struct {
	server *mockfs.MockServer
}

func (d *docBuilder) addDocument(path string, obj map[string]*pb.Value) {
	longPath := "projects/projectID/databases/(default)/documents/" + path
	d.server.AddRPC(&pb.BatchGetDocumentsRequest{
		Database:            "projects/projectID/databases/(default)",
		Documents:           []string{longPath},
		Mask:                nil,
		ConsistencySelector: nil,
	}, []any{
		&pb.BatchGetDocumentsResponse{
			Result: &pb.BatchGetDocumentsResponse_Found{
				Found: &pb.Document{
					Name:       longPath,
					CreateTime: timestamppb.Now(),
					UpdateTime: timestamppb.Now(),
					Fields:     obj,
				},
			},
			ReadTime:    timestamppb.New(time.Now()),
			Transaction: nil,
		},
	})
}

func (d *docBuilder) deleteDocument(path string) {
	longPath := "projects/projectID/databases/(default)/documents/" + path
	d.server.AddRPC(&pb.CommitRequest{
		Database: "projects/projectID/databases/(default)",
		Writes: []*pb.Write{{
			Operation:        &pb.Write_Delete{Delete: longPath},
			UpdateMask:       nil,
			UpdateTransforms: nil,
			CurrentDocument:  nil,
		}},
		Transaction: nil,
	}, &pb.CommitResponse{
		WriteResults: []*pb.WriteResult{{
			UpdateTime:       timestamppb.Now(),
			TransformResults: nil,
		}},
		CommitTime: timestamppb.Now(),
	})
}

func (d *docBuilder) setDocument(path string, obj map[string]*pb.Value) {
	longPath := "projects/projectID/databases/(default)/documents/" + path
	d.server.AddRPC(&pb.CommitRequest{
		Database: "projects/projectID/databases/(default)",
		Writes: []*pb.Write{{
			Operation: &pb.Write_Update{Update: &pb.Document{
				Name:       longPath,
				Fields:     obj,
				CreateTime: nil,
				UpdateTime: nil,
			}},
			UpdateMask:       nil,
			UpdateTransforms: nil,
			CurrentDocument:  nil,
		}},
		Transaction: nil,
	}, &pb.CommitResponse{
		WriteResults: []*pb.WriteResult{{
			UpdateTime:       timestamppb.Now(),
			TransformResults: nil,
		}},
		CommitTime: timestamppb.Now(),
	})
}

func (d *docBuilder) createDocumentTimestamps(
	path string,
	obj map[string]*pb.Value,
	timestamps []string,
) {
	longPath := "projects/projectID/databases/(default)/documents/" + path
	d.server.AddRPCAdjust(&pb.CommitRequest{
		Database: "projects/projectID/databases/(default)",
		Writes: []*pb.Write{{
			Operation: &pb.Write_Update{Update: &pb.Document{
				Name:       longPath,
				Fields:     obj,
				CreateTime: nil,
				UpdateTime: nil,
			}},
			UpdateMask:       nil,
			UpdateTransforms: nil,
			CurrentDocument:  nil,
		}},
		Transaction: nil,
	}, &pb.CommitResponse{
		WriteResults: []*pb.WriteResult{{
			UpdateTime:       timestamppb.Now(),
			TransformResults: nil,
		}},
		CommitTime: timestamppb.Now(),
	}, func(gotReq proto.Message) {
		//nolint:forcetypeassert
		update := gotReq.(*pb.CommitRequest).Writes[0].Operation.(*pb.Write_Update).Update
		for _, s := range timestamps {
			validateTimestamps(update.Fields, s)
		}
		update.Name = "projects/projectID/databases/(default)/documents/" + path
		//nolint:forcetypeassert
		gotReq.(*pb.CommitRequest).Writes[0].CurrentDocument = nil
	})
}

//nolint:gochecknoglobals
var notZeroTimestamp = &pb.Value_TimestampValue{TimestampValue: timestamppb.New(time.Unix(1, 1).UTC())}

func validateTimestamps(obj map[string]*pb.Value, key string) {
	val := obj[key].GetTimestampValue()
	obj[key] = &pb.Value{ValueType: notZeroTimestamp}

	if val.AsTime().Before(time.Now().Add(-time.Minute)) || val.AsTime().After(time.Now()) {
		panic(fmt.Sprintf("invalid timestamp: %s", val.AsTime().String()))
	}
}

func TestFillStruct(t *testing.T) { //nolint:funlen
	t.Parallel()
	assert := assert.New(t)

	type TestFillStructData struct {
		ID               string `json:"-"`
		TestString       string `json:"testString"`
		name             string
		TestMissing      string `json:"-"`
		TestNilPtr       *string
		TestPtr          *string
		TestNilInterface any
		TestNilMap       map[string]int
		TestNilSlice     []int
		TestBytes        []byte
		TestTime         time.Time
		TestDocumentRef  *origfirestore.DocumentRef
		TestBool         bool
		TestInt          int
		TestUint         uint
		TestFloat        float64
		TestSlice        []int
		TestArray        [4]int
		TestStruct       struct {
			TestString string
		}
		TestMap       map[string]int
		TestInterface any
	}

	type TestDocument struct {
		firestore.Doc[TestFillStructData]
		//nolint:unused
		hello string
	}

	type RootDocument struct {
		firestore.Doc[struct{}]
		TestCollection firestore.Collection[TestDocument, TestFillStructData] `json:"test"`
	}

	client, server, err := mockfs.New()
	assert.NoError(err)

	builder := docBuilder{server}
	now := time.Now().UTC()
	builder.addDocument("test/test", map[string]*pb.Value{
		"testString":       {ValueType: &pb.Value_StringValue{StringValue: "hello"}},
		"TestNilPtr":       {ValueType: &pb.Value_NullValue{NullValue: structpb.NullValue_NULL_VALUE}},
		"TestPtr":          {ValueType: &pb.Value_StringValue{StringValue: "hello"}},
		"TestNilInterface": {ValueType: &pb.Value_NullValue{NullValue: structpb.NullValue_NULL_VALUE}},
		"TestNilMap":       {ValueType: &pb.Value_NullValue{NullValue: structpb.NullValue_NULL_VALUE}},
		"TestNilSlice":     {ValueType: &pb.Value_NullValue{NullValue: structpb.NullValue_NULL_VALUE}},
		"TestBytes":        {ValueType: &pb.Value_BytesValue{BytesValue: []byte("hello")}},
		"TestTime":         {ValueType: &pb.Value_TimestampValue{TimestampValue: timestamppb.New(now)}},
		"TestDocumentRef": {
			ValueType: &pb.Value_ReferenceValue{
				ReferenceValue: "projects/projectID/databases/(default)/documents/test/test",
			},
		},
		"TestBool":  {ValueType: &pb.Value_BooleanValue{BooleanValue: true}},
		"TestInt":   {ValueType: &pb.Value_IntegerValue{IntegerValue: 1}},
		"TestUint":  {ValueType: &pb.Value_IntegerValue{IntegerValue: 1}},
		"TestFloat": {ValueType: &pb.Value_DoubleValue{DoubleValue: 1.0}},
		"TestSlice": {
			ValueType: &pb.Value_ArrayValue{
				ArrayValue: &pb.ArrayValue{Values: []*pb.Value{{ValueType: &pb.Value_IntegerValue{IntegerValue: 1}}}},
			},
		},
		"TestArray": {
			ValueType: &pb.Value_ArrayValue{
				ArrayValue: &pb.ArrayValue{Values: []*pb.Value{{ValueType: &pb.Value_IntegerValue{IntegerValue: 1}}}},
			},
		},
		"TestStruct": {
			ValueType: &pb.Value_MapValue{MapValue: &pb.MapValue{Fields: map[string]*pb.Value{
				"TestString": {ValueType: &pb.Value_StringValue{StringValue: "hello"}},
			}}},
		},
		"TestMap": {
			ValueType: &pb.Value_MapValue{MapValue: &pb.MapValue{Fields: map[string]*pb.Value{
				"TestString": {ValueType: &pb.Value_IntegerValue{IntegerValue: 1}},
			}}},
		},
		"TestInterface": {ValueType: &pb.Value_StringValue{StringValue: "hello"}},
	})

	resultStruct := TestFillStructData{
		ID:               "",
		TestString:       "hello",
		name:             "",
		TestMissing:      "",
		TestNilPtr:       nil,
		TestPtr:          &[]string{"hello"}[0],
		TestNilInterface: nil,
		TestNilMap:       make(map[string]int),
		TestNilSlice:     make([]int, 0),
		TestBytes:        []byte("hello"),
		TestTime:         now,
		TestDocumentRef:  client.Doc("test/test"),
		TestBool:         true,
		TestInt:          1,
		TestUint:         1,
		TestFloat:        1.0,
		TestSlice:        []int{1},
		TestArray:        [4]int{1, 0, 0, 0},
		TestStruct:       struct{ TestString string }{TestString: "hello"},
		TestMap:          map[string]int{"TestString": 1},
		TestInterface:    "hello",
	}

	builder.addDocument("test/test2", map[string]*pb.Value{"TestBool": {
		ValueType: &pb.Value_BooleanValue{BooleanValue: true},
	}})

	root := firestore.CreateRoot[RootDocument](client)
	data, err := root.TestCollection.ID("test").Data(ctx.FromBackground())
	assert.NoError(err)
	assert.Equal(resultStruct, data)

	_, err = root.TestCollection.ID("test2").Data(ctx.FromBackground())
	assert.Error(err)
}

func TestFillStructErrors(t *testing.T) { //nolint:funlen
	t.Parallel()
	assert := assert.New(t)

	type TestFillStructErrorsData struct {
		Invalid1 string         `json:"invalid1,omitempty"`
		Invalid2 *string        `json:"invalid2,omitempty"`
		Invalid3 fmt.Stringer   `json:"invalid3,omitempty"`
		Invalid4 int            `json:"invalid4,omitempty"`
		Invalid5 []int          `json:"invalid5,omitempty"`
		Invalid6 [1]int         `json:"invalid6,omitempty"`
		Invalid7 struct{}       `json:"invalid7,omitempty"`
		Invalid8 map[string]int `json:"invalid8,omitempty"`
	}

	type TestDocument struct {
		firestore.Doc[TestFillStructErrorsData]
	}

	type TestDocument2 struct {
		firestore.Doc[int]
	}

	type RootDocument struct {
		firestore.Doc[struct{}]
		TestCollection  firestore.Collection[TestDocument, TestFillStructErrorsData]  `json:"test"`
		TestCollection2 firestore.Collection[TestDocument2, TestFillStructErrorsData] `json:"test2"`
	}

	client, server, err := mockfs.New()
	assert.NoError(err)

	root := firestore.CreateRoot[RootDocument](client)
	builder := docBuilder{server}

	expectError := func(value map[string]*pb.Value) {
		builder.addDocument("test/test", value)

		_, err := root.TestCollection.ID("test").Data(ctx.FromBackground())
		assert.Error(err)
	}

	expectError(map[string]*pb.Value{
		"invalid1": {ValueType: &pb.Value_NullValue{NullValue: structpb.NullValue_NULL_VALUE}},
	})
	expectError(map[string]*pb.Value{
		"invalid2": {ValueType: &pb.Value_BytesValue{BytesValue: []byte("hello")}},
	})
	expectError(map[string]*pb.Value{
		"invalid1": {ValueType: &pb.Value_TimestampValue{TimestampValue: timestamppb.Now()}},
	})
	expectError(map[string]*pb.Value{
		"invalid1": {ValueType: &pb.Value_ReferenceValue{
			ReferenceValue: "projects/projectID/databases/(default)/documents/test/test",
		}},
	})
	expectError(map[string]*pb.Value{
		"invalid1": {ValueType: &pb.Value_ReferenceValue{
			ReferenceValue: "projects/projectID/databases/(default)/documents/test/test",
		}},
	})
	expectError(map[string]*pb.Value{
		"invalid1": {ValueType: &pb.Value_BooleanValue{BooleanValue: true}},
	})
	expectError(map[string]*pb.Value{
		"invalid4": {ValueType: &pb.Value_StringValue{StringValue: "hello"}},
	})
	expectError(map[string]*pb.Value{
		"invalid1": {ValueType: &pb.Value_IntegerValue{IntegerValue: 1}},
	})
	expectError(map[string]*pb.Value{
		"invalid1": {ValueType: &pb.Value_DoubleValue{DoubleValue: 1.0}},
	})
	expectError(map[string]*pb.Value{
		"invalid5": {ValueType: &pb.Value_ArrayValue{ArrayValue: &pb.ArrayValue{Values: []*pb.Value{
			{ValueType: &pb.Value_StringValue{StringValue: "hello"}},
		}}}},
	})
	expectError(map[string]*pb.Value{
		"invalid6": {ValueType: &pb.Value_ArrayValue{ArrayValue: &pb.ArrayValue{Values: []*pb.Value{
			{ValueType: &pb.Value_StringValue{StringValue: "hello"}},
		}}}},
	})
	expectError(map[string]*pb.Value{
		"invalid1": {ValueType: &pb.Value_ArrayValue{ArrayValue: &pb.ArrayValue{Values: []*pb.Value{
			{ValueType: &pb.Value_StringValue{StringValue: "hello"}},
		}}}},
	})
	expectError(map[string]*pb.Value{
		"invalid7": {ValueType: &pb.Value_MapValue{MapValue: &pb.MapValue{Fields: map[string]*pb.Value{
			"hello": {ValueType: &pb.Value_StringValue{StringValue: "hello"}},
		}}}},
	})
	expectError(map[string]*pb.Value{
		"invalid8": {ValueType: &pb.Value_MapValue{MapValue: &pb.MapValue{Fields: map[string]*pb.Value{
			"hello": {ValueType: &pb.Value_StringValue{StringValue: "hello"}},
		}}}},
	})
	expectError(map[string]*pb.Value{
		"invalid1": {ValueType: &pb.Value_MapValue{MapValue: &pb.MapValue{Fields: map[string]*pb.Value{
			"hello": {ValueType: &pb.Value_StringValue{StringValue: "hello"}},
		}}}},
	})
	expectError(map[string]*pb.Value{
		"invalid2": {ValueType: &pb.Value_GeoPointValue{GeoPointValue: &latlng.LatLng{Latitude: 1.0, Longitude: 1.0}}},
	})
	expectError(map[string]*pb.Value{
		"invalid3": {ValueType: &pb.Value_GeoPointValue{GeoPointValue: &latlng.LatLng{Latitude: 1.0, Longitude: 1.0}}},
	})

	builder.addDocument("test2/test", map[string]*pb.Value{
		"invalid1": {ValueType: &pb.Value_StringValue{StringValue: "hello"}},
	})

	_, err = root.TestCollection2.ID("test").Data(ctx.FromBackground())
	assert.Error(err)
}

type testDocumentData struct {
	Version    string `json:"version"`
	TestString string `json:"testString"`
}

func (t testDocumentData) CurrentVersion() string {
	return defVersion
}

func (t testDocumentData) Migrate(m map[string]any) map[string]any {
	m["version"] = defVersion

	return m
}

func TestFillStructMigration(t *testing.T) { //nolint:funlen
	t.Parallel()
	assert := assert.New(t)

	type TestDocument struct {
		firestore.Doc[testDocumentData]
	}

	type RootDocument struct {
		firestore.Doc[struct{}]
		TestCollection firestore.Collection[TestDocument, testDocumentData] `json:"test"`
	}

	client, server, err := mockfs.New()
	assert.NoError(err)

	builder := docBuilder{server}
	builder.addDocument("test/test", map[string]*pb.Value{
		"testString": {ValueType: &pb.Value_StringValue{StringValue: "hello"}},
		"version":    {ValueType: &pb.Value_StringValue{StringValue: defVersion}},
	})
	builder.addDocument("test/test2", map[string]*pb.Value{
		"testString": {ValueType: &pb.Value_StringValue{StringValue: "hello"}},
		"version":    {ValueType: &pb.Value_StringValue{StringValue: "100"}},
	})
	builder.addDocument("test/test3", map[string]*pb.Value{
		"testString": {ValueType: &pb.Value_StringValue{StringValue: "hello"}},
		"version":    {ValueType: &pb.Value_StringValue{StringValue: "0.0.0"}},
	})
	builder.addDocument("test/test4", map[string]*pb.Value{
		"testString": {ValueType: &pb.Value_NullValue{NullValue: structpb.NullValue_NULL_VALUE}},
		"version":    {ValueType: &pb.Value_StringValue{StringValue: "0.0.0"}},
	})
	server.AddRPC(&pb.BatchGetDocumentsRequest{
		Database:            "projects/projectID/databases/(default)",
		Documents:           []string{"projects/projectID/databases/(default)/documents/test/test5"},
		Mask:                nil,
		ConsistencySelector: nil,
	}, []any{&pb.BatchGetDocumentsResponse{
		Result:      nil,
		ReadTime:    timestamppb.New(time.Now()),
		Transaction: nil,
	}})

	resultStruct := testDocumentData{
		Version:    "1.0.0",
		TestString: "hello",
	}

	root := firestore.CreateRoot[RootDocument](client)
	data, err := root.TestCollection.ID("test").Data(ctx.FromBackground())
	assert.NoError(err)
	assert.Equal(resultStruct, data)

	_, err = root.TestCollection.ID("test2").Data(ctx.FromBackground())
	assert.Error(err)

	data, err = root.TestCollection.ID("test3").Data(ctx.FromBackground())
	assert.NoError(err)
	assert.Equal(resultStruct, data)

	_, err = root.TestCollection.ID("test4").Data(ctx.FromBackground())
	assert.Error(err)

	_, err = root.TestCollection.ID("test5").Data(ctx.FromBackground())
	assert.Error(err)
}

type testTransformStruct struct {
	Version      string `json:"version"`
	hello        string
	Unexported   string `json:"unexported,omitempty"`
	TestNilPtr   *string
	TestPtr      *string
	TestBytes    []byte
	TestDocRef   *origfirestore.DocumentRef
	TestBool     bool
	TestInt      int
	TestUint     uint
	TestFloat    float64
	TestArray    [4]int
	TestNilSlice []int
	TestSlice    []int
	TestNilMap   map[string]int
	TestMap      map[string]int
	TestStruct   struct {
		TestString string
	}
	TestInterface any
}

func (t testTransformStruct) CurrentVersion() string {
	return "1.0.0"
}

func TestTransformStruct(t *testing.T) { //nolint:funlen
	t.Parallel()

	assert := assert.New(t)

	type TestDocument struct {
		firestore.Doc[testTransformStruct]
	}

	type RootDocument struct {
		firestore.Doc[struct{}]

		TestCollection firestore.Collection[TestDocument, testTransformStruct] `json:"test"`
	}

	client, server, err := mockfs.New()
	assert.NoError(err)

	builder := docBuilder{server}
	builder.setDocument("test/test", map[string]*pb.Value{
		"version":    {ValueType: &pb.Value_StringValue{StringValue: "1.0.0"}},
		"TestNilPtr": {ValueType: &pb.Value_NullValue{NullValue: structpb.NullValue_NULL_VALUE}},
		"TestPtr":    {ValueType: &pb.Value_StringValue{StringValue: "hello"}},
		"TestBytes":  {ValueType: &pb.Value_BytesValue{BytesValue: []byte("hello")}},
		"TestDocRef": {
			ValueType: &pb.Value_ReferenceValue{
				ReferenceValue: "projects/projectID/databases/(default)/documents/test/test",
			},
		},
		"TestBool":  {ValueType: &pb.Value_BooleanValue{BooleanValue: true}},
		"TestInt":   {ValueType: &pb.Value_IntegerValue{IntegerValue: 1}},
		"TestUint":  {ValueType: &pb.Value_IntegerValue{IntegerValue: 1}},
		"TestFloat": {ValueType: &pb.Value_DoubleValue{DoubleValue: 1.0}},
		"TestArray": {
			ValueType: &pb.Value_ArrayValue{
				ArrayValue: &pb.ArrayValue{Values: []*pb.Value{
					{ValueType: &pb.Value_IntegerValue{IntegerValue: 1}},
					{ValueType: &pb.Value_IntegerValue{IntegerValue: 0}},
					{ValueType: &pb.Value_IntegerValue{IntegerValue: 0}},
					{ValueType: &pb.Value_IntegerValue{IntegerValue: 0}},
				}},
			},
		},
		"TestNilSlice": {
			ValueType: &pb.Value_ArrayValue{
				ArrayValue: &pb.ArrayValue{Values: []*pb.Value{}},
			},
		},
		"TestSlice": {
			ValueType: &pb.Value_ArrayValue{
				ArrayValue: &pb.ArrayValue{Values: []*pb.Value{{ValueType: &pb.Value_IntegerValue{IntegerValue: 1}}}},
			},
		},
		"TestNilMap": {
			ValueType: &pb.Value_MapValue{MapValue: &pb.MapValue{Fields: map[string]*pb.Value{}}},
		},
		"TestMap": {
			ValueType: &pb.Value_MapValue{MapValue: &pb.MapValue{Fields: map[string]*pb.Value{
				"TestString": {ValueType: &pb.Value_IntegerValue{IntegerValue: 1}},
			}}},
		},
		"TestStruct": {
			ValueType: &pb.Value_MapValue{MapValue: &pb.MapValue{Fields: map[string]*pb.Value{
				"TestString": {ValueType: &pb.Value_StringValue{StringValue: "hello"}},
			}}},
		},
		"TestInterface": {ValueType: &pb.Value_StringValue{StringValue: "hello"}},
	})

	root := firestore.CreateRoot[RootDocument](client)

	_, err = root.TestCollection.ID("test").Set(ctx.FromBackground(), testTransformStruct{
		Version:       "0.0.0",
		hello:         "",
		Unexported:    "",
		TestNilPtr:    nil,
		TestPtr:       &[]string{"hello"}[0],
		TestBytes:     []byte("hello"),
		TestDocRef:    client.Doc("test/test"),
		TestBool:      true,
		TestInt:       1,
		TestUint:      1,
		TestFloat:     1.0,
		TestArray:     [4]int{1, 0, 0, 0},
		TestNilSlice:  nil,
		TestSlice:     []int{1},
		TestNilMap:    nil,
		TestMap:       map[string]int{"TestString": 1},
		TestStruct:    struct{ TestString string }{TestString: "hello"},
		TestInterface: "hello",
	})
	assert.NoError(err)
}

func TestTransformStructErrors(t *testing.T) { //nolint:funlen
	t.Parallel()

	type testTransformStruct struct {
		InvalidPtr      *complex64           `json:"invalidPtr,omitempty"`
		InvalidArray    [1]complex64         `json:"invalidArray,omitempty"`
		InvalidSlice    []complex64          `json:"invalidSlice,omitempty"`
		InvalidMapKey   map[complex64]int    `json:"invalidMapKey,omitempty"`
		InvalidMapValue map[string]complex64 `json:"invalidMapValue,omitempty"`
		InvalidStruct   struct {
			InvalidField complex64 `json:"invalidField,omitempty"`
		} `json:"invalidStruct,omitempty"`
		InvalidInterface error `json:"invalidInterface,omitempty"`
	}

	type TestDocument struct {
		firestore.Doc[testTransformStruct]
	}

	type RootDocument struct {
		firestore.Doc[struct{}]

		TestCollection firestore.Collection[TestDocument, testTransformStruct] `json:"test"`
	}

	client, _, err := mockfs.New()
	assert.NoError(t, err)

	root := firestore.CreateRoot[RootDocument](client)

	buf := new(bytes.Buffer)
	noLogCtx := ctx.FromBackground().WithLog(log.New(log.WithWriter(buf)))

	_, err = root.TestCollection.ID("test").Set(noLogCtx, testTransformStruct{
		InvalidPtr:      &[]complex64{0}[0],
		InvalidArray:    [1]complex64{0},
		InvalidSlice:    nil,
		InvalidMapKey:   nil,
		InvalidMapValue: nil,
		InvalidStruct: struct {
			InvalidField complex64 `json:"invalidField,omitempty"`
		}{InvalidField: 0},
		InvalidInterface: nil,
	})
	assert.Error(t, err)

	_, err = root.TestCollection.ID("test").Set(noLogCtx, testTransformStruct{
		InvalidPtr:      nil,
		InvalidArray:    [1]complex64{1},
		InvalidSlice:    nil,
		InvalidMapKey:   nil,
		InvalidMapValue: nil,
		InvalidStruct: struct {
			InvalidField complex64 `json:"invalidField,omitempty"`
		}{InvalidField: 0},
		InvalidInterface: nil,
	})
	assert.Error(t, err)

	_, err = root.TestCollection.ID("test").Set(noLogCtx, testTransformStruct{
		InvalidPtr:      nil,
		InvalidArray:    [1]complex64{0},
		InvalidSlice:    []complex64{0},
		InvalidMapKey:   nil,
		InvalidMapValue: nil,
		InvalidStruct: struct {
			InvalidField complex64 `json:"invalidField,omitempty"`
		}{InvalidField: 0},
		InvalidInterface: nil,
	})
	assert.Error(t, err)

	_, err = root.TestCollection.ID("test").Set(noLogCtx, testTransformStruct{
		InvalidPtr:      nil,
		InvalidArray:    [1]complex64{0},
		InvalidSlice:    nil,
		InvalidMapKey:   make(map[complex64]int),
		InvalidMapValue: nil,
		InvalidStruct: struct {
			InvalidField complex64 `json:"invalidField,omitempty"`
		}{InvalidField: 0},
		InvalidInterface: nil,
	})
	assert.Error(t, err)

	_, err = root.TestCollection.ID("test").Set(noLogCtx, testTransformStruct{
		InvalidPtr:      nil,
		InvalidArray:    [1]complex64{0},
		InvalidSlice:    nil,
		InvalidMapKey:   nil,
		InvalidMapValue: map[string]complex64{"hello": 1},
		InvalidStruct: struct {
			InvalidField complex64 `json:"invalidField,omitempty"`
		}{InvalidField: 0},
		InvalidInterface: nil,
	})
	assert.Error(t, err)

	_, err = root.TestCollection.ID("test").Set(noLogCtx, testTransformStruct{
		InvalidPtr:      nil,
		InvalidArray:    [1]complex64{0},
		InvalidSlice:    nil,
		InvalidMapKey:   nil,
		InvalidMapValue: nil,
		InvalidStruct: struct {
			InvalidField complex64 `json:"invalidField,omitempty"`
		}{InvalidField: 1},
		InvalidInterface: nil,
	})
	assert.Error(t, err)

	_, err = root.TestCollection.ID("test").Set(noLogCtx, testTransformStruct{
		InvalidPtr:      nil,
		InvalidArray:    [1]complex64{0},
		InvalidSlice:    nil,
		InvalidMapKey:   nil,
		InvalidMapValue: nil,
		InvalidStruct: struct {
			InvalidField complex64 `json:"invalidField,omitempty"`
		}{InvalidField: 0},
		//nolint:goerr113
		InvalidInterface: errors.New("hello"),
	})
	assert.Error(t, err)
}

type testTimeStruct struct {
	UpdatedAt time.Time `json:"updatedAt"`
	CreatedAt time.Time `json:"createdAt"`
}

func TestTimeStruct(t *testing.T) {
	t.Parallel()

	assert := assert.New(t)

	type TestDocument struct {
		firestore.Doc[testTimeStruct]

		Nested firestore.Collection[TestDocument, testTimeStruct] `json:"nested"`
	}

	type RootDocument struct {
		firestore.Doc[struct{}]

		TestCollection  firestore.Collection[TestDocument, testTimeStruct] `json:"test"`
		TestCollection2 firestore.Collection[TestDocument, int]            `json:"test2"`
	}

	client, server, err := mockfs.New()
	assert.NoError(err)

	builder := docBuilder{server}
	builder.createDocumentTimestamps("test/test", map[string]*pb.Value{
		"updatedAt": {ValueType: notZeroTimestamp},
		"createdAt": {ValueType: notZeroTimestamp},
	}, []string{"createdAt", "updatedAt"})

	root := firestore.CreateRoot[RootDocument](client)

	_, _, err = root.TestCollection.Add(ctx.FromBackground(), testTimeStruct{
		UpdatedAt: time.Time{},
		CreatedAt: time.Time{},
	})
	assert.NoError(err)

	builder.setDocument("test/test", map[string]*pb.Value{
		"updatedAt": {ValueType: notZeroTimestamp},
		"createdAt": {ValueType: notZeroTimestamp},
	})

	buf := new(bytes.Buffer)
	ctxNoLog := ctx.FromBackground().WithLog(log.New(log.WithWriter(buf)))
	_, _, err = root.TestCollection.Add(ctxNoLog, testTimeStruct{
		UpdatedAt: time.Time{},
		CreatedAt: time.Time{},
	})
	assert.Error(err)

	_, _, err = root.TestCollection2.Add(ctxNoLog, 1)
	assert.Error(err)

	builder.createDocumentTimestamps("test/test/nested/test", map[string]*pb.Value{
		"updatedAt": {ValueType: notZeroTimestamp},
		"createdAt": {ValueType: notZeroTimestamp},
	}, []string{"createdAt", "updatedAt"})

	_, _, err = root.TestCollection.ID("test").Nested.Add(ctx.FromBackground(), testTimeStruct{
		UpdatedAt: time.Time{},
		CreatedAt: time.Time{},
	})
	assert.NoError(err)
}

func TestDelete(t *testing.T) {
	t.Parallel()

	assert := assert.New(t)

	type TestDocument struct {
		firestore.Doc[testTransformStruct]
	}

	type RootDocument struct {
		firestore.Doc[struct{}]

		TestCollection firestore.Collection[TestDocument, testTransformStruct] `json:"test"`
	}

	client, server, err := mockfs.New()
	assert.NoError(err)

	builder := docBuilder{server}
	builder.deleteDocument("test/test")

	root := firestore.CreateRoot[RootDocument](client)
	_, err = root.TestCollection.ID("test").Delete(ctx.FromBackground())
	assert.NoError(err)
}

func TestTransaction(t *testing.T) { //nolint:funlen
	t.Parallel()

	assert := assert.New(t)

	type testTransactionStruct struct{}

	type TestDocument struct {
		firestore.Doc[testTransactionStruct]
	}

	type RootDocument struct {
		firestore.Doc[struct{}]

		TestCollection firestore.Collection[TestDocument, testTransactionStruct] `json:"test"`
	}

	client, server, err := mockfs.New()
	assert.NoError(err)

	// builder := docBuilder{server}

	server.AddRPC(&pb.BeginTransactionRequest{
		Database: "projects/projectID/databases/(default)",
		Options:  nil,
	}, &pb.BeginTransactionResponse{
		Transaction: []byte(""),
	})
	server.AddRPC(&pb.BatchGetDocumentsRequest{
		Database:            "projects/projectID/databases/(default)",
		Documents:           []string{"projects/projectID/databases/(default)/documents/test/test"},
		Mask:                nil,
		ConsistencySelector: nil,
	}, []any{&pb.BatchGetDocumentsResponse{
		Result: &pb.BatchGetDocumentsResponse_Found{Found: &pb.Document{
			Name:       "projects/projectID/databases/(default)/documents/test/test",
			Fields:     map[string]*pb.Value{},
			CreateTime: timestamppb.Now(),
			UpdateTime: timestamppb.Now(),
		}},
		Transaction: []byte(""),
		ReadTime:    timestamppb.Now(),
	}})
	server.AddRPC(&pb.CommitRequest{
		Database: "projects/projectID/databases/(default)",
		Writes: []*pb.Write{{
			Operation: &pb.Write_Update{
				Update: &pb.Document{
					Name:       "projects/projectID/databases/(default)/documents/test/test",
					Fields:     nil,
					CreateTime: nil,
					UpdateTime: nil,
				},
			},
			UpdateMask:       nil,
			UpdateTransforms: nil,
			CurrentDocument:  nil,
		}},
		Transaction: []byte(""),
	}, &pb.CommitResponse{}) //nolint:exhaustruct

	root := firestore.CreateRoot[RootDocument](client)

	err = root.TestCollection.ID("test").Trans(ctx.FromBackground(),
		func(testTransactionStruct) (testTransactionStruct, error) {
			return testTransactionStruct{}, nil
		},
	)
	assert.NoError(err)
}

func TestTransactionError(t *testing.T) { //nolint:funlen
	t.Parallel()

	assert := assert.New(t)

	type testTransactionStruct struct {
		InvalidField complex64 `json:"invalidField,omitempty"`
	}

	type TestDocument struct {
		firestore.Doc[testTransactionStruct]
	}

	type RootDocument struct {
		firestore.Doc[struct{}]

		TestCollection firestore.Collection[TestDocument, testTransactionStruct] `json:"test"`
	}

	client, server, err := mockfs.New()
	assert.NoError(err)

	server.AddRPC(&pb.BeginTransactionRequest{
		Database: "projects/projectID/databases/(default)",
		Options:  nil,
	}, &pb.BeginTransactionResponse{
		Transaction: []byte(""),
	})
	server.AddRPC(&pb.BatchGetDocumentsRequest{
		Database:            "projects/projectID/databases/(default)",
		Documents:           []string{"projects/projectID/databases/(default)/documents/test/test"},
		Mask:                nil,
		ConsistencySelector: nil,
	}, []any{&pb.BatchGetDocumentsResponse{
		Result: &pb.BatchGetDocumentsResponse_Missing{
			Missing: "projects/projectID/databases/(default)/documents/test/test",
		},
		Transaction: []byte(""),
		ReadTime:    timestamppb.Now(),
	}})
	server.AddRPC(&pb.RollbackRequest{
		Database:    "projects/projectID/databases/(default)",
		Transaction: []byte(""),
	}, &emptypb.Empty{})

	root := firestore.CreateRoot[RootDocument](client)

	buf := new(bytes.Buffer)
	noLogCtx := ctx.FromBackground().WithLog(log.New(log.WithWriter(buf)))

	err = root.TestCollection.ID("test").Trans(noLogCtx,
		func(testTransactionStruct) (testTransactionStruct, error) {
			return testTransactionStruct{
				InvalidField: 0,
			}, nil
		},
	)
	assert.Error(err)

	server.AddRPC(&pb.BeginTransactionRequest{
		Database: "projects/projectID/databases/(default)",
		Options:  nil,
	}, &pb.BeginTransactionResponse{
		Transaction: []byte(""),
	})
	server.AddRPC(&pb.BatchGetDocumentsRequest{
		Database:            "projects/projectID/databases/(default)",
		Documents:           []string{"projects/projectID/databases/(default)/documents/test/test"},
		Mask:                nil,
		ConsistencySelector: nil,
	}, []any{&pb.BatchGetDocumentsResponse{
		Result: &pb.BatchGetDocumentsResponse_Found{Found: &pb.Document{
			Name:       "projects/projectID/databases/(default)/documents/test/test",
			Fields:     map[string]*pb.Value{},
			CreateTime: timestamppb.Now(),
			UpdateTime: timestamppb.Now(),
		}},
		Transaction: []byte(""),
		ReadTime:    timestamppb.Now(),
	}})
	server.AddRPC(&pb.RollbackRequest{
		Database:    "projects/projectID/databases/(default)",
		Transaction: []byte(""),
	}, &emptypb.Empty{})

	err = root.TestCollection.ID("test").Trans(noLogCtx,
		func(testTransactionStruct) (testTransactionStruct, error) {
			//nolint:goerr113
			return testTransactionStruct{}, errors.New("hello")
		},
	)
	assert.Error(err)

	server.AddRPC(&pb.BeginTransactionRequest{
		Database: "projects/projectID/databases/(default)",
		Options:  nil,
	}, &pb.BeginTransactionResponse{
		Transaction: []byte(""),
	})
	server.AddRPC(&pb.BatchGetDocumentsRequest{
		Database:            "projects/projectID/databases/(default)",
		Documents:           []string{"projects/projectID/databases/(default)/documents/test/test"},
		Mask:                nil,
		ConsistencySelector: nil,
	}, []any{&pb.BatchGetDocumentsResponse{
		Result: &pb.BatchGetDocumentsResponse_Found{Found: &pb.Document{
			Name:       "projects/projectID/databases/(default)/documents/test/test",
			Fields:     map[string]*pb.Value{},
			CreateTime: timestamppb.Now(),
			UpdateTime: timestamppb.Now(),
		}},
		Transaction: []byte(""),
		ReadTime:    timestamppb.Now(),
	}})
	server.AddRPC(&pb.RollbackRequest{
		Database:    "projects/projectID/databases/(default)",
		Transaction: []byte(""),
	}, &emptypb.Empty{})

	err = root.TestCollection.ID("test").Trans(noLogCtx,
		func(testTransactionStruct) (testTransactionStruct, error) {
			return testTransactionStruct{InvalidField: 1}, nil
		},
	)
	assert.Error(err)
}

func TestDocuments(t *testing.T) { //nolint:funlen
	t.Parallel()

	assert := assert.New(t)

	type TestDocument struct {
		firestore.Doc[struct{}]
	}

	type RootDocument struct {
		firestore.Doc[struct{}]

		TestCollection firestore.Collection[TestDocument, struct{}] `json:"test"`
	}

	client, server, err := mockfs.New()
	assert.NoError(err)

	root := firestore.CreateRoot[RootDocument](client)

	server.AddRPC(&pb.ListDocumentsRequest{
		Parent:       "projects/projectID/databases/(default)/documents",
		CollectionId: "test",
		PageSize:     0,
		PageToken:    "",
		OrderBy:      "",
		Mask: &pb.DocumentMask{
			FieldPaths: nil,
		},
		ConsistencySelector: nil,
		ShowMissing:         true,
	}, &pb.ListDocumentsResponse{
		Documents: []*pb.Document{
			{
				Name:       "projects/projectID/databases/(default)/documents/test/test",
				Fields:     make(map[string]*pb.Value),
				CreateTime: timestamppb.Now(),
				UpdateTime: timestamppb.Now(),
			},
		},
		NextPageToken: "",
	})

	docs, err := root.TestCollection.Documents(ctx.FromBackground())
	assert.NoError(err)
	assert.Len(docs, 1)

	server.AddRPC(&pb.ListDocumentsRequest{
		Parent:       "projects/projectID/databases/(default)/documents",
		CollectionId: "test",
		PageSize:     0,
		PageToken:    "",
		OrderBy:      "",
		Mask: &pb.DocumentMask{
			FieldPaths: nil,
		},
		ConsistencySelector: nil,
		ShowMissing:         true,
	}, &pb.ListDocumentsResponse{
		Documents: []*pb.Document{
			{
				Name:       "projects/projectID/databases/(default)/documents/test/test",
				Fields:     make(map[string]*pb.Value),
				CreateTime: timestamppb.Now(),
				UpdateTime: timestamppb.Now(),
			},
			{
				Name:       "projects/projectID/databases/(default)/documents/test/test",
				Fields:     make(map[string]*pb.Value),
				CreateTime: timestamppb.Now(),
				UpdateTime: timestamppb.Now(),
			},
		},
		NextPageToken: "",
	})

	iter := root.TestCollection.DocumentsIter(ctx.FromBackground())
	assert.True(iter.HasNext())
	assert.True(iter.HasNext())

	_, err = iter.Next()
	assert.NoError(err)

	_, err = iter.Next()
	assert.NoError(err)

	_, err = iter.Next()
	assert.Error(err)

	assert.False(iter.HasNext())

	_, err = iter.Next()
	assert.Error(err)

	server.AddRPC(&pb.ListDocumentsRequest{
		Parent:       "projects/projectID/databases/(default)/documents",
		CollectionId: "test",
		PageSize:     0,
		PageToken:    "",
		OrderBy:      "",
		Mask: &pb.DocumentMask{
			FieldPaths: nil,
		},
		ConsistencySelector: nil,
		ShowMissing:         true,
	}, &pb.ListDocumentsResponse{
		Documents: []*pb.Document{
			{
				Name:       "projects/projectID/databases/(default)",
				Fields:     make(map[string]*pb.Value),
				CreateTime: timestamppb.Now(),
				UpdateTime: timestamppb.Now(),
			},
		},
		NextPageToken: "",
	})

	ctxNoLog := ctx.FromBackground().WithLog(log.New(log.WithWriter(new(bytes.Buffer))))

	_, err = root.TestCollection.Documents(ctxNoLog)
	assert.Error(err)
}

func TestQueryDocuments(t *testing.T) { //nolint:funlen
	t.Parallel()

	assert := assert.New(t)

	type testQueryDocumentData struct {
		TestString string `json:"test"`
	}

	type TestDocument struct {
		firestore.Doc[testQueryDocumentData]
	}

	type RootDocument struct {
		firestore.Doc[testQueryDocumentData]

		TestCollection firestore.Collection[TestDocument, testQueryDocumentData] `json:"test"`
	}

	client, server, err := mockfs.New()
	assert.NoError(err)

	root := firestore.CreateRoot[RootDocument](client)

	server.AddRPC(&pb.RunQueryRequest{
		Parent: "projects/projectID/databases/(default)/documents",
		QueryType: &pb.RunQueryRequest_StructuredQuery{
			//nolint:exhaustruct
			StructuredQuery: &pb.StructuredQuery{
				From: []*pb.StructuredQuery_CollectionSelector{
					{
						CollectionId:   "test",
						AllDescendants: false,
					},
				},
			},
		},
		ConsistencySelector: nil,
	}, []any{&pb.RunQueryResponse{
		Transaction: []byte(""),
		Document: &pb.Document{
			Name: "projects/projectID/databases/(default)/documents/test/test",
			Fields: map[string]*pb.Value{
				"test": {ValueType: &pb.Value_StringValue{StringValue: "hello"}},
			},
			CreateTime: timestamppb.Now(),
			UpdateTime: timestamppb.Now(),
		},
		ReadTime:             timestamppb.Now(),
		SkippedResults:       0,
		ContinuationSelector: &pb.RunQueryResponse_Done{Done: true},
	}})

	data, err := root.TestCollection.Query(ctx.FromBackground())
	assert.NoError(err)
	assert.Equal(testQueryDocumentData{TestString: "hello"}, data[0])

	server.AddRPC(&pb.RunQueryRequest{
		Parent: "projects/projectID/databases/(default)/documents",
		QueryType: &pb.RunQueryRequest_StructuredQuery{
			//nolint:exhaustruct
			StructuredQuery: &pb.StructuredQuery{
				From: []*pb.StructuredQuery_CollectionSelector{
					{
						CollectionId:   "test",
						AllDescendants: false,
					},
				},
			},
		},
		ConsistencySelector: nil,
	}, []any{&pb.RunQueryResponse{
		Transaction: []byte(""),
		Document: &pb.Document{
			Name: "projects/projectID/databases/(default)/documents/test/test",
			Fields: map[string]*pb.Value{
				"test2": {ValueType: &pb.Value_StringValue{StringValue: "hello"}},
			},
			CreateTime: timestamppb.Now(),
			UpdateTime: timestamppb.Now(),
		},
		ReadTime:             timestamppb.Now(),
		SkippedResults:       0,
		ContinuationSelector: &pb.RunQueryResponse_Done{Done: true},
	}})

	noLogCtx := ctx.FromBackground().WithLog(log.New(log.WithWriter(new(bytes.Buffer))))

	_, err = root.TestCollection.Query(noLogCtx)
	assert.Error(err)

	server.AddRPC(&pb.RunQueryRequest{
		Parent: "projects/projectID/databases/(default)/documents",
		QueryType: &pb.RunQueryRequest_StructuredQuery{
			//nolint:exhaustruct
			StructuredQuery: &pb.StructuredQuery{
				From: []*pb.StructuredQuery_CollectionSelector{
					{
						CollectionId:   "test",
						AllDescendants: false,
					},
				},
			},
		},
		ConsistencySelector: nil,
	}, []any{&pb.RunQueryResponse{
		Transaction: []byte(""),
		Document: &pb.Document{
			Name: "projects/projectID/databases/(default)/documents/test/test1",
			Fields: map[string]*pb.Value{
				"test": {ValueType: &pb.Value_StringValue{StringValue: "hello"}},
			},
			CreateTime: timestamppb.Now(),
			UpdateTime: timestamppb.Now(),
		},
		ReadTime:             timestamppb.Now(),
		SkippedResults:       0,
		ContinuationSelector: &pb.RunQueryResponse_Done{Done: true},
	}, &pb.RunQueryResponse{
		Transaction: []byte(""),
		Document: &pb.Document{
			Name: "projects/projectID/databases/(default)/documents/test/test2",
			Fields: map[string]*pb.Value{
				"test": {ValueType: &pb.Value_StringValue{StringValue: "hello"}},
			},
			CreateTime: timestamppb.Now(),
			UpdateTime: timestamppb.Now(),
		},
		ReadTime:             timestamppb.Now(),
		SkippedResults:       0,
		ContinuationSelector: &pb.RunQueryResponse_Done{Done: true},
	}})

	iter := root.TestCollection.QueryIter(noLogCtx, func(query firestore.Query) firestore.Query {
		return query
	})
	assert.True(iter.HasNext())
	assert.True(iter.HasNext())

	_, err = iter.Next()
	assert.NoError(err)

	_, err = iter.Next()
	assert.NoError(err)

	_, err = iter.Next()
	assert.Error(err)

	assert.False(iter.HasNext())

	_, err = iter.Next()
	assert.Error(err)
}
*/
