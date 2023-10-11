package jsonschema_test

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/DimmyJing/valise/jsonschema"
	"github.com/DimmyJing/valise/log"
	"github.com/stretchr/testify/assert"
)

type TestEnum string

const (
	TestEnumA TestEnum = "A"
	TestEnumB TestEnum = "B"
)

func (e TestEnum) Members() []string {
	return jsonschema.EnumMembers(TestEnumA, TestEnumB)
}

//nolint:tagliatelle
type TestSchema struct {
	TestBool      bool
	TestInt       int
	TestInt8      int8
	TestInt16     int16
	TestInt32     int32
	TestUInt      uint
	TestUInt8     uint8
	TestUInt16    uint16
	TestUInt32    uint32
	TestInt64     int64
	TestUInt64    uint64
	TestFloat32   float32
	TestFloat64   float64
	TestArray     [8]int
	TestInterface any
	TestMap       map[string]int
	TestPtr       *string
	TestSlice     []int
	TestString    string
	TestEnum      TestEnum
	TestStruct    struct {
		TestBool bool
	}
	//nolint:unused
	testUnexported bool
	TestUnexported bool   `json:"-"`
	TestFieldName  string `json:"testFieldName2"`
	TestOptional   string `json:",omitempty"`
	TestTime       time.Time
}

func TestSchemaReflect(t *testing.T) {
	t.Parallel()

	//nolint:exhaustruct
	schema, err := jsonschema.AnyToSchema(reflect.TypeOf(TestSchema{}))
	assert.NoError(t, err)

	log.SetDefault(log.New(log.WithCharm()))

	res, err := json.MarshalIndent(schema, "", "\t")
	assert.NoError(t, err)

	file, err := os.Open("testdata/reflect.json")
	assert.NoError(t, err)

	fileContent, err := io.ReadAll(file)
	assert.NoError(t, err)

	assert.Equal(t, string(fileContent), string(res)+"\n")

	var schema2 jsonschema.JSONSchema
	err = json.Unmarshal(res, &schema2)
	assert.NoError(t, err)
}

func TestSchemaError(t *testing.T) {
	t.Parallel()

	type invalidArray [1]complex64

	_, err := jsonschema.AnyToSchema(reflect.TypeOf(invalidArray{}))
	assert.Error(t, err)

	_, err = jsonschema.AnyToSchema(reflect.TypeOf((*fmt.Stringer)(nil)))
	assert.Error(t, err)

	_, err = jsonschema.AnyToSchema(reflect.TypeOf(map[string]complex64{}))
	assert.Error(t, err)

	_, err = jsonschema.AnyToSchema(reflect.TypeOf([]complex64{}))
	assert.Error(t, err)

	//nolint:tagliatelle
	_, err = jsonschema.AnyToSchema(reflect.TypeOf(struct {
		LowerCase  string `json:"lowerCase"`
		Underscore string `json:"_hello"`
		Field      complex64
	}{
		LowerCase:  "",
		Underscore: "",
		Field:      0,
	}))
	assert.Error(t, err)
}

type TestSchema2 struct {
	//nolint:unused
	hello string
	Hello string
	//nolint:tagalign
	Hello2 string `json:"hello5" in:"path"`
	Hello3 string `json:"-"`
	Hello4 string `json:"hello4,omitempty"`
}

type TestSchema3 struct {
	Hello string `in:"hello"`
}

type TestSchema4 struct {
	Hello complex64
}

func TestParamReflect(t *testing.T) {
	t.Parallel()

	//nolint:exhaustruct
	schema, err := jsonschema.ParametersToSchema(reflect.TypeOf(TestSchema2{}))
	assert.NoError(t, err)
	assert.Equal(t, "hello", schema[0].Name)
	assert.Equal(t, "query", schema[0].In)
	assert.Equal(t, true, schema[0].Required)
	assert.Equal(t, "string", schema[0].Schema.Type)

	assert.Equal(t, "hello5", schema[1].Name)
	assert.Equal(t, "path", schema[1].In)
	assert.Equal(t, true, schema[1].Required)
	assert.Equal(t, "string", schema[1].Schema.Type)

	assert.Equal(t, "hello4", schema[2].Name)
	assert.Equal(t, "query", schema[2].In)
	assert.Equal(t, false, schema[2].Required)
	assert.Equal(t, "string", schema[2].Schema.Type)

	//nolint:exhaustruct
	_, err = jsonschema.ParametersToSchema(reflect.TypeOf(TestSchema3{}))
	assert.Error(t, err)

	//nolint:exhaustruct
	_, err = jsonschema.ParametersToSchema(reflect.TypeOf(TestSchema4{}))
	assert.Error(t, err)
}
