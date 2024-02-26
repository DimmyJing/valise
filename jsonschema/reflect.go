package jsonschema

import (
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"time"

	orderedmap "github.com/wk8/go-ordered-map/v2"
)

func EnumMembers[M ~string](members ...M) []string {
	res := make([]string, len(members))
	for i, m := range members {
		res[i] = string(m)
	}

	return res
}

type EnumMember interface {
	Members() []string
}

//nolint:gochecknoglobals
var enumInterface = reflect.TypeOf((*EnumMember)(nil)).Elem()

var errReflectType = errors.New("invalid reflect.Type")

func convertType(value reflect.Type) (*JSONSchema, error) { //nolint:funlen,gocognit,gocyclo,cyclop
	var schema JSONSchema

	if desc, found := getDescription(value.PkgPath(), value.Name(), ""); found {
		schema.Description = desc
	}

	switch value.Kind() {
	case reflect.Bool:
		//nolint:goconst
		schema.Type = "boolean"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32:
		//nolint:goconst
		schema.Type = "integer"
		schema.Format = "int32"
	case reflect.Int64, reflect.Uint64:
		schema.Type = "integer"
		schema.Format = "int64"
	case reflect.Float32:
		//nolint:goconst
		schema.Type = "number"
		schema.Format = "float"
	case reflect.Float64:
		schema.Type = "number"
		schema.Format = "double"
	case reflect.Array:
		//nolint:goconst
		schema.Type = "array"

		val, err := convertType(value.Elem())
		if err != nil {
			return nil, fmt.Errorf("error converting array element type: %w", err)
		}

		schema.Items = val
		schema.MaxItems = value.Len()
		schema.MinItems = value.Len()
	case reflect.Interface:
		if value.NumMethod() == 0 {
			schema.boolean = new(bool)
			*schema.boolean = true
		} else {
			return nil, fmt.Errorf("invalid reflect type %s: %w", value.Kind().String(), errReflectType)
		}
	case reflect.Map:
		//nolint:goconst
		schema.Type = "object"

		val, err := convertType(value.Elem())
		if err != nil {
			return nil, fmt.Errorf("error converting map element type: %w", err)
		}

		schema.AdditionalProperties = val
	case reflect.Ptr:
		val, err := convertType(value.Elem())
		if err != nil {
			return nil, fmt.Errorf("error converting pointer type: %w", err)
		}

		return val, nil
	case reflect.Slice:
		if value.Elem().Kind() == reflect.Uint8 {
			//nolint:goconst
			schema.Type = "string"
			schema.Format = "binary"
		} else {
			schema.Type = "array"

			val, err := convertType(value.Elem())
			if err != nil {
				return nil, fmt.Errorf("error converting slice element type: %w", err)
			}

			schema.Items = val
		}
	case reflect.String:
		schema.Type = "string"

		if value.Implements(enumInterface) {
			if member, ok := reflect.Zero(value).Interface().(EnumMember); ok {
				members := member.Members()
				schema.Enums = members
			}
		}
	case reflect.Struct:
		//nolint:nestif
		if value == reflect.TypeOf(time.Time{}) {
			schema.Type = "string"
			schema.Format = "date-time"
		} else {
			schema.Type = "object"

			for i := 0; i < value.NumField(); i++ {
				field := value.Field(i)
				if !field.IsExported() {
					continue
				}

				fieldName := strings.ToLower(string(field.Name[0])) + field.Name[1:]
				optional := false

				if jsonTag, found := field.Tag.Lookup("json"); found {
					splitTags := strings.Split(jsonTag, ",")
					if len(splitTags) > 0 {
						if splitTags[0] == "-" && len(splitTags) == 1 {
							continue
						} else if splitTags[0] != "" {
							fieldName = splitTags[0]
						}
					}

					if slices.Contains(splitTags[1:], "omitempty") {
						optional = true
					}
				}

				if schema.Properties == nil {
					schema.Properties = orderedmap.New[string, *JSONSchema]()
				}

				property, err := convertType(field.Type)
				if err != nil {
					return nil, fmt.Errorf("error converting struct field %s: %w", fieldName, err)
				}

				if desc, found := getDescription(value.PkgPath(), value.Name(), field.Name); found {
					property.Description = desc
				}

				property.Title = fieldName

				schema.Properties.Set(fieldName, property)

				if !optional {
					schema.Required = append(schema.Required, fieldName)
				}
			}

			schema.AdditionalProperties = &JSONSchemaFalse
		}
	case reflect.Invalid, reflect.Uintptr, reflect.Complex64, reflect.Complex128,
		reflect.Chan, reflect.Func, reflect.UnsafePointer:
		return nil, fmt.Errorf("invalid reflect type %s: %w", value.Kind().String(), errReflectType)
	}

	if schema.Title == "" {
		schema.Title = value.Name()
	}

	return &schema, nil
}

func AnyToSchema(value reflect.Type) (*JSONSchema, error) {
	return convertType(value)
}

var errInvalidTag = errors.New("invalid tag")

func RequestBodyToSchema(value reflect.Type) (*JSONSchema, error) { //nolint:funlen,cyclop
	if value.Kind() != reflect.Struct {
		return nil, fmt.Errorf("invalid reflect type %s, expected struct: %w", value.Kind().String(), errReflectType)
	}

	var schema JSONSchema

	if desc, found := getDescription(value.PkgPath(), value.Name(), ""); found {
		schema.Description = desc
	}

	schema.Title = value.Name()
	schema.Type = "object"

	for i := 0; i < value.NumField(); i++ {
		field := value.Field(i)
		if !field.IsExported() {
			continue
		}

		fieldName := strings.ToLower(string(field.Name[0])) + field.Name[1:]
		optional := false

		//nolint:nestif
		if jsonTag, found := field.Tag.Lookup("json"); found {
			splitTags := strings.Split(jsonTag, ",")
			if len(splitTags) > 0 {
				if splitTags[0] == "-" && len(splitTags) == 1 {
					continue
				} else if splitTags[0] != "" {
					fieldName = splitTags[0]
				}
			}

			if slices.Contains(splitTags[1:], "omitempty") {
				optional = true
			}
		}

		if _, found := field.Tag.Lookup("in"); found {
			continue
		}

		if schema.Properties == nil {
			schema.Properties = orderedmap.New[string, *JSONSchema]()
		}

		property, err := AnyToSchema(field.Type)
		if err != nil {
			return nil, fmt.Errorf("error converting struct field %s: %w", fieldName, err)
		}

		if desc, found := getDescription(value.PkgPath(), value.Name(), field.Name); found {
			property.Description = desc
		}

		property.Title = fieldName

		schema.Properties.Set(fieldName, property)

		if !optional {
			schema.Required = append(schema.Required, fieldName)
		}
	}

	schema.AdditionalProperties = &JSONSchemaFalse

	return &schema, nil
}

func ParametersToSchema(value reflect.Type, defaultToQuery bool) ([]OpenAPIParameter, error) { //nolint:cyclop
	params := []OpenAPIParameter{}

	for i := 0; i < value.NumField(); i++ {
		field := value.Field(i)
		if !field.IsExported() {
			continue
		}

		fieldName := strings.ToLower(string(field.Name[0])) + field.Name[1:]
		optional := false
		paramIn := "query"

		//nolint:nestif
		if jsonTag, found := field.Tag.Lookup("json"); found {
			splitTags := strings.Split(jsonTag, ",")
			if len(splitTags) > 0 {
				if splitTags[0] == "-" && len(splitTags) == 1 {
					continue
				} else if splitTags[0] != "" {
					fieldName = splitTags[0]
				}
			}

			if slices.Contains(splitTags[1:], "omitempty") {
				optional = true
			}
		}

		if inTag, found := field.Tag.Lookup("in"); found {
			if inTag != "path" && inTag != "query" {
				return nil, fmt.Errorf("invalid value for in tag %s: %w", inTag, errInvalidTag)
			}

			paramIn = inTag
		} else if !defaultToQuery {
			continue
		}

		property, err := convertType(field.Type)
		if err != nil {
			return nil, fmt.Errorf("error converting struct field %s: %w", fieldName, err)
		}

		desc, _ := getDescription(value.PkgPath(), value.Name(), field.Name)
		property.Description = desc

		property.Title = fieldName

		params = append(params, OpenAPIParameter{
			Schema:      property,
			Name:        fieldName,
			In:          paramIn,
			Description: property.Description,
			Required:    !optional,
		})
	}

	return params, nil
}
