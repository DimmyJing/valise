package rpc

import (
	"strings"

	"github.com/DimmyJing/valise/jsonschema"
	orderedmap "github.com/wk8/go-ordered-map/v2"
	"google.golang.org/protobuf/proto"
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
	Schema jsonschema.JSONInfo `json:"schema"`
}

type openAPIResponse struct {
	Description string                      `json:"description,omitempty"`
	Content     map[string]openAPIMediaType `json:"content"`
}

func (o *openAPIObject) addOperation( //nolint:funlen
	path string,
	input proto.Message,
	output proto.Message,
	method string,
	description string,
	tags []string,
) {
	operation := openAPIOperation{
		Tags:        tags,
		Description: description,
		Responses: map[string]openAPIResponse{
			"200": {
				Description: jsonschema.GetComment(output),
				Content: map[string]openAPIMediaType{
					"application/json": {Schema: *jsonschema.GenerateSchema(output)},
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

	//nolint:nestif
	if method == "get" || method == "delete" {
		parameters := []openAPIParameter{}

		schemas := jsonschema.GenerateSchemas(input)
		for _, schema := range schemas {
			parameters = append(parameters, openAPIParameter{
				openAPIMediaType: openAPIMediaType{Schema: schema.Schema},
				Name:             schema.Title,
				In:               "query",
				Description:      schema.Description,
				Required:         schema.Required,
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
			Description: jsonschema.GetComment(input),
			Required:    true,
			Content:     map[string]openAPIMediaType{"application/json": {Schema: *jsonschema.GenerateSchema(input)}},
		}}

		if method == "post" {
			schemaPath.Post = bodyOp
		} else if method == "put" {
			schemaPath.Put = bodyOp
		}
	}

	o.Paths.Set(path, schemaPath)
}
