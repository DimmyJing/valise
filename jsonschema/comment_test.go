package jsonschema_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/DimmyJing/valise/jsonschema"
	"github.com/DimmyJing/valise/jsonschema/testdata"
	"github.com/stretchr/testify/assert"
)

func TestComment(t *testing.T) {
	t.Parallel()

	err := jsonschema.InitCommentMap("../", "github.com/DimmyJing/valise")
	assert.NoError(t, err)

	var data testdata.Comment1

	schema, err := jsonschema.AnyToSchema(reflect.ValueOf(data).Type())
	assert.NoError(t, err)

	schemaBuf, err := json.MarshalIndent(schema, "", "  ")
	assert.NoError(t, err)

	result := `{
  "title": "Comment1",
  "description": "Comment1.",
  "type": "object",
  "properties": {
    "comment2": {
      "title": "comment2",
      "description": "Comment2",
      "type": "string"
    },
    "comment3": {
      "title": "comment3",
      "description": "Comment3",
      "type": "object",
      "properties": {
        "comment11": {
          "title": "comment11",
          "description": "Comment11",
          "type": "string"
        },
        "comment12": {
          "title": "comment12",
          "description": "Comment12",
          "type": "string"
        }
      },
      "required": [
        "comment11",
        "comment12"
      ],
      "additionalProperties": false
    },
    "comment4": {
      "title": "comment4",
      "description": "Comment4",
      "type": "string"
    }
  },
  "required": [
    "comment2",
    "comment3",
    "comment4"
  ],
  "additionalProperties": false
}`
	assert.Equal(t, result, string(schemaBuf))
}
