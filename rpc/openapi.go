package rpc

import (
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/DimmyJing/valise/jsonschema"
	orderedmap "github.com/wk8/go-ordered-map/v2"
)

type openAPIObject struct {
	Openapi string                                          `json:"openapi"`
	Info    openAPIInfo                                     `json:"info"`
	Paths   *orderedmap.OrderedMap[string, openAPIPathItem] `json:"paths"`
}

type openAPIInfo struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Version     string `json:"version"`
}

type openAPIPathItem struct {
	Get    *openAPIParamOperation `json:"get,omitempty"`
	Post   *openAPIBodyOperation  `json:"post,omitempty"`
	Put    *openAPIBodyOperation  `json:"put,omitempty"`
	Delete *openAPIParamOperation `json:"delete,omitempty"`
}

type openAPIOperation struct {
	Tags        []string                   `json:"tags,omitempty"`
	Description string                     `json:"description,omitempty"`
	Responses   map[string]openAPIResponse `json:"responses"`
}

type openAPIParamOperation struct {
	openAPIOperation
	Parameters []openAPIParameter `json:"parameters,omitempty"`
}

type openAPIBodyOperation struct {
	openAPIOperation
	RequestBody openAPIRequestBody `json:"requestBody,omitempty"`
}

type openAPIParameter struct {
	openAPIMediaType
	Name        string `json:"name"`
	In          string `json:"in"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

type openAPIRequestBody struct {
	Description string                      `json:"description,omitempty"`
	Content     map[string]openAPIMediaType `json:"content"`
	Required    bool                        `json:"required,omitempty"`
}

type openAPIMediaType struct {
	Schema jsonschema.JSONSchema `json:"schema"`
}

type openAPIResponse struct {
	Description string                      `json:"description,omitempty"`
	Content     map[string]openAPIMediaType `json:"content"`
}

func (o *openAPIObject) addOperation( //nolint:funlen
	path string,
	input reflect.Type,
	output reflect.Type,
	method string,
	description string,
	tags []string,
) error {
	outSchema, err := jsonschema.AnyToSchema(output)
	if err != nil {
		return fmt.Errorf("failed to generate schema for output: %w", err)
	}

	operation := openAPIOperation{
		Tags:        tags,
		Description: description,
		Responses: map[string]openAPIResponse{
			"200": {
				Description: outSchema.Description,
				Content: map[string]openAPIMediaType{
					"application/json": {Schema: *outSchema},
				},
			},
		},
	}

	schemaPath := openAPIPathItem{
		Get:    nil,
		Post:   nil,
		Put:    nil,
		Delete: nil,
	}

	method = strings.ToLower(method)

	inputSchema, err := jsonschema.AnyToSchema(input)
	if err != nil {
		return fmt.Errorf("failed to generate schema for input: %w", err)
	}

	//nolint:nestif
	if method == "get" || method == "delete" {
		parameters := []openAPIParameter{}

		for pair := inputSchema.Properties.Oldest(); pair != nil; pair = pair.Next() {
			parameters = append(parameters, openAPIParameter{
				openAPIMediaType: openAPIMediaType{Schema: *pair.Value},
				Name:             pair.Key,
				In:               "query",
				Description:      pair.Value.Description,
				Required:         slices.Contains(inputSchema.Required, pair.Key),
			})
		}

		paramOp := &openAPIParamOperation{openAPIOperation: operation, Parameters: parameters}

		if method == "get" {
			schemaPath.Get = paramOp
		} else if method == "delete" {
			schemaPath.Delete = paramOp
		}
	} else {
		bodyOp := &openAPIBodyOperation{openAPIOperation: operation, RequestBody: openAPIRequestBody{
			Description: inputSchema.Description,
			Required:    true,
			Content:     map[string]openAPIMediaType{"application/json": {Schema: *inputSchema}},
		}}

		if method == "post" {
			schemaPath.Post = bodyOp
		} else if method == "put" {
			schemaPath.Put = bodyOp
		}
	}

	o.Paths.Set(path, schemaPath)

	return nil
}
