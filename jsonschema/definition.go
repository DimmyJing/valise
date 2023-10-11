package jsonschema

import (
	"encoding/json"

	orderedmap "github.com/wk8/go-ordered-map/v2"
)

type JSONSchema struct {
	Title                string                                      `json:"title,omitempty"`
	Description          string                                      `json:"description,omitempty"`
	Type                 string                                      `json:"type,omitempty"`
	Format               string                                      `json:"format,omitempty"`
	Enums                []string                                    `json:"enum,omitempty"`
	Items                *JSONSchema                                 `json:"items,omitempty"`
	MaxItems             int                                         `json:"maxItems,omitempty"`
	MinItems             int                                         `json:"minItems,omitempty"`
	Properties           *orderedmap.OrderedMap[string, *JSONSchema] `json:"properties,omitempty"`
	Required             []string                                    `json:"required,omitempty"`
	AdditionalProperties *JSONSchema                                 `json:"additionalProperties,omitempty"`
	boolean              *bool
}

type OpenAPIParameter struct {
	Schema      *JSONSchema `json:"schema"`
	Name        string      `json:"name"`
	In          string      `json:"in"`
	Description string      `json:"description,omitempty"`
	Required    bool        `json:"required,omitempty"`
}

//nolint:gochecknoglobals,exhaustruct
var (
	boolean         = true
	JSONSchemaTrue  = JSONSchema{boolean: &boolean}
	JSONSchemaFalse = JSONSchema{boolean: new(bool)}
)

func (s *JSONSchema) MarshalJSON() ([]byte, error) {
	type Alias JSONSchema

	if s.boolean != nil {
		if *s.boolean {
			return []byte("true"), nil
		} else {
			return []byte("false"), nil
		}
	}

	//nolint:wrapcheck
	return json.Marshal((*Alias)(s))
}

func (s *JSONSchema) UnmarshalJSON(data []byte) error {
	type Alias JSONSchema

	strData := string(data)

	switch strData {
	case "true":
		s.boolean = new(bool)
		*s.boolean = true
	case "false":
		s.boolean = new(bool)
	default:
		//nolint:wrapcheck
		return json.Unmarshal(data, (*Alias)(s))
	}

	return nil
}
