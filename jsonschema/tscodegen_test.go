package jsonschema_test

import (
	"testing"

	"github.com/DimmyJing/valise/jsonschema"
	"github.com/stretchr/testify/assert"
	orderedmap "github.com/wk8/go-ordered-map/v2"
)

func TestFormatComment(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "/** hello */\n", jsonschema.FormatComment("hello"))
	assert.Equal(t, "", jsonschema.FormatComment(""))
	assert.Equal(t, "/**\n * a\n * b\n */\n", jsonschema.FormatComment("a\nb"))
}

//nolint:exhaustruct
func TestSchemaConvert(t *testing.T) { //nolint:funlen
	t.Parallel()

	schema := &jsonschema.JSONSchema{
		Type:                 "object",
		AdditionalProperties: &jsonschema.JSONSchemaFalse,
		Properties:           orderedmap.New[string, *jsonschema.JSONSchema](),
	}

	schema.Properties.Set("test1", &jsonschema.JSONSchema{
		Type: "",
	})
	schema.Properties.Set("test2", &jsonschema.JSONSchema{
		Type:  "string",
		Enums: []string{"a", "b"},
	})
	schema.Properties.Set("test3", &jsonschema.JSONSchema{
		Type:   "string",
		Format: "date-time",
	})
	schema.Properties.Set("test4", &jsonschema.JSONSchema{
		Type: "string",
	})
	schema.Properties.Set("test5", &jsonschema.JSONSchema{
		Type: "integer",
	})
	schema.Properties.Set("test6", &jsonschema.JSONSchema{
		Type: "boolean",
	})
	schema.Properties.Set("test7", &jsonschema.JSONSchema{
		Type: "array",
		Items: &jsonschema.JSONSchema{
			Type: "string",
		},
	})
	schema.Properties.Set("test8", &jsonschema.JSONSchema{
		Type: "null",
	})
	schema.Properties.Set("test9", &jsonschema.JSONSchema{
		Type:                 "object",
		AdditionalProperties: &jsonschema.JSONSchemaTrue,
	})
	schema.Properties.Set("test10", &jsonschema.JSONSchema{
		Type:                 "object",
		AdditionalProperties: &jsonschema.JSONSchema{Type: "string"},
	})
	schema.Properties.Set("test11", &jsonschema.JSONSchema{
		Type:                 "object",
		AdditionalProperties: &jsonschema.JSONSchemaFalse,
		Properties:           orderedmap.New[string, *jsonschema.JSONSchema](),
	})

	schema.Required = []string{"test1"}

	res, err := jsonschema.JSONSchemaToTS(schema, "test")
	assert.NoError(t, err)

	expected := `test{
  test1: unknown;
  test2?: "a" | "b";
  test3?: Date;
  test4?: string;
  test5?: number;
  test6?: boolean;
  test7?: string[];
  test8?: null;
  test9?: Record<string, any>;
  test10?: Record<string, string>;
  test11?: Record<string, never>;
}`
	assert.Equal(t, expected, res)
}

//nolint:exhaustruct
func TestSchemaConvertError(t *testing.T) {
	t.Parallel()

	properties := orderedmap.New[string, *jsonschema.JSONSchema]()
	properties.Set("test", &jsonschema.JSONSchema{
		Type:                 "object",
		AdditionalProperties: &jsonschema.JSONSchema{Type: "invalid"},
	})

	schema := &jsonschema.JSONSchema{
		Type: "array",
		Items: &jsonschema.JSONSchema{
			Type:                 "object",
			Properties:           properties,
			AdditionalProperties: &jsonschema.JSONSchemaFalse,
		},
	}

	_, err := jsonschema.JSONSchemaToTS(schema, "test")
	assert.Error(t, err)
}
