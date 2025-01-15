package jsonschema_test

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/DimmyJing/valise/jsonschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type customMarshaller struct {
	Value string
}

func (c customMarshaller) MarshalJSON() ([]byte, error) {
	return []byte("custom" + c.Value + "custom"), nil
}

var _ json.Marshaler = customMarshaller{Value: ""}

//nolint:forcetypeassert
func TestValueToAny(t *testing.T) { //nolint:funlen
	t.Parallel()

	//nolint:tagliatelle
	type testVal struct {
		TestBool          bool
		TestInt           int
		TestUint          uint
		TestFloat         float64
		TestArray         [8]int
		TestInterface     any
		TestNilInterface  any
		TestPtrInterface  any
		TestMap           map[string]int
		TestPtr           *string
		TestNilPtr        *string
		TestSlice         []int
		TestString        string
		TestBytes         []byte
		TestTime          time.Time
		TestIgnore        string `json:"-"`
		TestName          string `json:"testCustomName"`
		TestOptional      string `json:",omitempty"`
		TestOptional2     string `json:",omitempty"`
		TestJSONRaw       json.RawMessage
		TestCustomMarshal customMarshaller
		testNotExported   string
	}

	input := testVal{
		TestBool:         true,
		TestInt:          1,
		TestUint:         2,
		TestFloat:        3.0,
		TestArray:        [8]int{1, 2, 3, 4, 5, 6, 7, 8},
		TestInterface:    "test",
		TestNilInterface: nil,
		TestPtrInterface: &[]string{"hello"}[0],
		TestMap: map[string]int{
			"test": 1,
		},
		TestPtr:       &[]string{"hello"}[0],
		TestNilPtr:    nil,
		TestSlice:     []int{1, 2, 3, 4, 5, 6, 7, 8},
		TestString:    "test",
		TestBytes:     []byte("test"),
		TestTime:      time.Unix(1, 1).UTC(),
		TestIgnore:    "test",
		TestName:      "test",
		TestOptional:  "test",
		TestOptional2: "",
		TestJSONRaw:   json.RawMessage(`{"test": "test"}`),
		TestCustomMarshal: customMarshaller{
			Value: "test",
		},
		testNotExported: "test",
	}

	res, err := jsonschema.ValueToAny(reflect.ValueOf(input))
	resMap := res.(map[string]any)

	require.NoError(t, err)
	assert.Equal(t, true, resMap["testBool"])
	assert.Equal(t, int64(1), resMap["testInt"])
	assert.Equal(t, uint64(2), resMap["testUint"])
	assert.InEpsilon(t, 3.0, resMap["testFloat"], 0.0001)
	assert.Equal(t,
		[]any{int64(1), int64(2), int64(3), int64(4), int64(5), int64(6), int64(7), int64(8)},
		resMap["testArray"],
	)
	assert.Equal(t, "test", resMap["testInterface"])
	assert.Nil(t, resMap["testNilInterface"])
	assert.Equal(t, "hello", resMap["testPtrInterface"])
	assert.Equal(t, map[string]any{"test": int64(1)}, resMap["testMap"])
	assert.Equal(t, "hello", resMap["testPtr"])
	assert.Nil(t, resMap["testNilPtr"])
	assert.Equal(t,
		[]any{int64(1), int64(2), int64(3), int64(4), int64(5), int64(6), int64(7), int64(8)},
		resMap["testSlice"],
	)
	assert.Equal(t, "test", resMap["testString"])
	assert.Equal(t, []byte("test"), resMap["testBytes"])
	assert.Equal(t, time.Unix(1, 1).UTC(), resMap["testTime"])
	assert.Equal(t, "test", resMap["testCustomName"])
	assert.Equal(t, "test", resMap["testOptional"])
	assert.Nil(t, resMap["testNotExported"])
	assert.Nil(t, resMap["testOptional2"])
	msg, ok := resMap["testJSONRaw"].(json.RawMessage)
	assert.True(t, ok)
	//nolint:testifylint
	assert.Equal(t, json.RawMessage(`{"test": "test"}`), msg)
	assert.Equal(t, customMarshaller{Value: "test"}, resMap["testCustomMarshal"])
}

func TestValueToAnyError(t *testing.T) {
	t.Parallel()

	_, err := jsonschema.ValueToAny(reflect.ValueOf(complex64(1)))
	assert.Error(t, err)

	_, err = jsonschema.ValueToAny(reflect.ValueOf([]complex64{0}))
	assert.Error(t, err)

	_, err = jsonschema.ValueToAny(reflect.ValueOf(map[string]complex64{"hello": 1}))
	assert.Error(t, err)

	_, err = jsonschema.ValueToAny(reflect.ValueOf(map[complex64]string{}))
	assert.Error(t, err)

	_, err = jsonschema.ValueToAny(reflect.ValueOf(&[]complex64{0}[0]))
	assert.Error(t, err)

	_, err = jsonschema.ValueToAny(reflect.ValueOf(struct{ A complex64 }{A: 1}))
	assert.Error(t, err)

	_, err = jsonschema.ValueToAny(reflect.ValueOf([1]complex64{1}))
	assert.Error(t, err)

	var stringer fmt.Stringer = time.Time{}
	_, err = jsonschema.ValueToAny(reflect.ValueOf(&stringer))
	assert.Error(t, err)
}

func TestAnyToValue(t *testing.T) { //nolint:funlen
	t.Parallel()

	//nolint:tagliatelle
	type testVal struct {
		TestBool         bool
		TestBool2        bool
		TestBool3        bool
		TestInt          int
		TestInt2         int
		TestInt3         int
		TestUint         uint
		TestUint2        uint
		TestUint3        uint
		TestFloat        float64
		TestFloat2       float64
		TestFloat3       float64
		TestArray        [8]int
		TestArray2       [8]int
		TestInterface    any
		TestNilInterface any
		TestPtrInterface any
		TestMap          map[string]int
		TestNilMap       map[string]int
		TestMissingMap   map[string]int `json:",omitempty"`
		TestPtr          *string
		TestNilPtr       *string
		TestSlice        []int
		TestSlice2       []int
		TestNilSlice     []int
		TestMissingSlice []int `json:",omitempty"`
		TestString       string
		TestString2      string
		TestBytes        []byte
		TestTime         time.Time
		TestIgnore       string `json:"-"`
		TestName         string `json:"testCustomName"`
		TestOptional     string `json:",omitempty"`
		TestOptional2    string `json:",omitempty"`
		TestEnum         TestEnum
		testNotExported  string
	}

	input := map[string]any{
		"testBool":         true,
		"testBool2":        "true",
		"testBool3":        []string{"true"},
		"testInt":          int64(1),
		"testInt2":         "1",
		"testInt3":         []string{"1"},
		"testUint":         uint64(2),
		"testUint2":        "2",
		"testUint3":        []string{"2"},
		"testFloat":        3.0,
		"testFloat2":       "3.0",
		"testFloat3":       []string{"3.0"},
		"testArray":        []any{int64(1), int64(2), int64(3), int64(4), int64(5), int64(6), int64(7), int64(8)},
		"testArray2":       []string{"1", "2", "3", "4", "5", "6", "7", "8"},
		"testInterface":    "test",
		"testNilInterface": nil,
		"testPtrInterface": "hello",
		"testMap":          map[string]any{"test": int64(1)},
		"testNilMap":       nil,
		"testPtr":          "hello",
		"testNilPtr":       nil,
		"testNilSlice":     nil,
		"testSlice":        []any{int64(1), int64(2), int64(3), int64(4), int64(5), int64(6), int64(7), int64(8)},
		"testSlice2":       []string{"1", "2", "3", "4", "5", "6", "7", "8"},
		"testString":       "test",
		"testString2":      []string{"test"},
		"testBytes":        []byte("test"),
		"testTime":         time.Unix(1, 1).UTC(),
		"testCustomName":   "test",
		"testOptional":     "test",
		"testEnum":         "A",
	}

	var val testVal
	err := jsonschema.AnyToValue(input, reflect.ValueOf(&val).Elem())
	require.NoError(t, err)

	assert.True(t, val.TestBool)
	assert.Equal(t, 1, val.TestInt)
	assert.Equal(t, uint(2), val.TestUint)
	assert.InEpsilon(t, 3.0, val.TestFloat, 0.0001)
	assert.Equal(t, [8]int{1, 2, 3, 4, 5, 6, 7, 8}, val.TestArray)
	assert.Equal(t, "test", val.TestInterface)
	assert.Nil(t, val.TestNilInterface)
	assert.Equal(t, "hello", val.TestPtrInterface)
	assert.Equal(t, map[string]int{"test": 1}, val.TestMap)
	assert.Equal(t, map[string]int{}, val.TestNilMap)
	assert.Equal(t, map[string]int(nil), val.TestMissingMap)
	assert.Equal(t, &[]string{"hello"}[0], val.TestPtr)
	assert.Equal(t, (*string)(nil), val.TestNilPtr)
	assert.Equal(t, []int{}, val.TestNilSlice)
	assert.Equal(t, []int{1, 2, 3, 4, 5, 6, 7, 8}, val.TestSlice)
	assert.Equal(t, []int{}, val.TestNilSlice)
	assert.Equal(t, []int(nil), val.TestMissingSlice)
	assert.Equal(t, "test", val.TestString)
	assert.Equal(t, []byte("test"), val.TestBytes)
	assert.Equal(t, time.Unix(1, 1).UTC(), val.TestTime)
	assert.Equal(t, "test", val.TestName)
	assert.Equal(t, "test", val.TestOptional)
	assert.Equal(t, "", val.TestOptional2)
	assert.Equal(t, TestEnumA, val.TestEnum)
	assert.Equal(t, "", val.testNotExported)
	assert.True(t, val.TestBool2)
	assert.Equal(t, 1, val.TestInt2)
	assert.Equal(t, uint(2), val.TestUint2)
	assert.InEpsilon(t, 3.0, val.TestFloat2, 0.0001)
	assert.Equal(t, [8]int{1, 2, 3, 4, 5, 6, 7, 8}, val.TestArray2)
	assert.Equal(t, []int{1, 2, 3, 4, 5, 6, 7, 8}, val.TestSlice2)
	assert.True(t, val.TestBool3)
	assert.Equal(t, 1, val.TestInt3)
	assert.Equal(t, uint(2), val.TestUint3)
	assert.InEpsilon(t, 3.0, val.TestFloat3, 0.0001)
	assert.Equal(t, "test", val.TestString2)
}

func TestAnyToValueError(t *testing.T) { //nolint:funlen
	t.Parallel()

	err := jsonschema.AnyToValue(1, reflect.ValueOf(1))
	require.Error(t, err)

	var valBool bool
	err = jsonschema.AnyToValue(1, reflect.ValueOf(&valBool).Elem())
	require.Error(t, err)

	err = jsonschema.AnyToValue("hello", reflect.ValueOf(&valBool).Elem())
	require.Error(t, err)

	err = jsonschema.AnyToValue([]string{"hello", "world"}, reflect.ValueOf(&valBool).Elem())
	require.Error(t, err)

	err = jsonschema.AnyToValue([]string{"hello"}, reflect.ValueOf(&valBool).Elem())
	require.Error(t, err)

	var valInt int
	err = jsonschema.AnyToValue(true, reflect.ValueOf(&valInt).Elem())
	require.Error(t, err)

	err = jsonschema.AnyToValue("a", reflect.ValueOf(&valInt).Elem())
	require.Error(t, err)

	err = jsonschema.AnyToValue([]string{"a", "b"}, reflect.ValueOf(&valInt).Elem())
	require.Error(t, err)

	err = jsonschema.AnyToValue([]string{"a"}, reflect.ValueOf(&valInt).Elem())
	require.Error(t, err)

	var valUint uint
	err = jsonschema.AnyToValue(true, reflect.ValueOf(&valUint).Elem())
	require.Error(t, err)

	err = jsonschema.AnyToValue("a", reflect.ValueOf(&valUint).Elem())
	require.Error(t, err)

	err = jsonschema.AnyToValue([]string{"a", "b"}, reflect.ValueOf(&valUint).Elem())
	require.Error(t, err)

	err = jsonschema.AnyToValue([]string{"a"}, reflect.ValueOf(&valUint).Elem())
	require.Error(t, err)

	var valFloat float64
	err = jsonschema.AnyToValue(true, reflect.ValueOf(&valFloat).Elem())
	require.Error(t, err)

	err = jsonschema.AnyToValue("a", reflect.ValueOf(&valFloat).Elem())
	require.Error(t, err)

	err = jsonschema.AnyToValue([]string{"a", "b"}, reflect.ValueOf(&valFloat).Elem())
	require.Error(t, err)

	err = jsonschema.AnyToValue([]string{"a"}, reflect.ValueOf(&valFloat).Elem())
	require.Error(t, err)

	var valArray [1]int
	err = jsonschema.AnyToValue(true, reflect.ValueOf(&valArray).Elem())
	require.Error(t, err)

	err = jsonschema.AnyToValue([]string{"a"}, reflect.ValueOf(&valArray).Elem())
	require.Error(t, err)

	err = jsonschema.AnyToValue([]string{"a", "b"}, reflect.ValueOf(&valArray).Elem())
	require.Error(t, err)

	var valArray2 [1]int
	err = jsonschema.AnyToValue([]any{}, reflect.ValueOf(&valArray2).Elem())
	require.Error(t, err)

	var valArray3 [1]int
	err = jsonschema.AnyToValue([]any{true}, reflect.ValueOf(&valArray3).Elem())
	require.Error(t, err)

	var stringer fmt.Stringer = time.Time{}
	err = jsonschema.AnyToValue(true, reflect.ValueOf(&stringer).Elem())
	require.Error(t, err)

	var valMap map[int]int
	err = jsonschema.AnyToValue(true, reflect.ValueOf(&valMap).Elem())
	require.Error(t, err)

	var valMap2 map[string]int
	err = jsonschema.AnyToValue(map[string]any{"a": true}, reflect.ValueOf(&valMap2).Elem())
	require.Error(t, err)

	var valMap3 map[string]int
	err = jsonschema.AnyToValue(true, reflect.ValueOf(&valMap3).Elem())
	require.Error(t, err)

	var valPtr *complex64
	err = jsonschema.AnyToValue(true, reflect.ValueOf(&valPtr).Elem())
	require.Error(t, err)

	var valBytes []byte
	err = jsonschema.AnyToValue(true, reflect.ValueOf(&valBytes).Elem())
	require.Error(t, err)

	var valSlice []int
	err = jsonschema.AnyToValue([]any{true}, reflect.ValueOf(&valSlice).Elem())
	require.Error(t, err)

	err = jsonschema.AnyToValue([]string{"a"}, reflect.ValueOf(&valSlice).Elem())
	require.Error(t, err)

	var valSlice2 []int
	err = jsonschema.AnyToValue(true, reflect.ValueOf(&valSlice2).Elem())
	require.Error(t, err)

	var valEnum TestEnum
	err = jsonschema.AnyToValue("C", reflect.ValueOf(&valEnum).Elem())
	require.Error(t, err)

	var valString string
	err = jsonschema.AnyToValue(true, reflect.ValueOf(&valString).Elem())
	require.Error(t, err)

	err = jsonschema.AnyToValue([]string{"hello", "world"}, reflect.ValueOf(&valString).Elem())
	require.Error(t, err)

	var valTime time.Time
	err = jsonschema.AnyToValue(true, reflect.ValueOf(&valTime).Elem())
	require.Error(t, err)

	var valStruct struct{ Test string }
	err = jsonschema.AnyToValue(map[string]any{"test": true}, reflect.ValueOf(&valStruct).Elem())
	require.Error(t, err)

	var valStruct2 struct{ Test string }
	err = jsonschema.AnyToValue(map[string]any{}, reflect.ValueOf(&valStruct2).Elem())
	require.Error(t, err)

	var valStruct3 struct{}
	err = jsonschema.AnyToValue(map[string]any{"test": true}, reflect.ValueOf(&valStruct3).Elem())
	require.Error(t, err)

	var valStruct4 struct{}
	err = jsonschema.AnyToValue(true, reflect.ValueOf(&valStruct4).Elem())
	require.Error(t, err)
}
