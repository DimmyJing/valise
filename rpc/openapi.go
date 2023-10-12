package rpc

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"regexp"
	"runtime"
	"slices"
	"strings"

	"github.com/DimmyJing/valise/ctx"
	"github.com/DimmyJing/valise/jsonschema"
	"github.com/labstack/echo/v4"
	orderedmap "github.com/wk8/go-ordered-map/v2"
)

type openAPIObject struct {
	OpenAPI string                                                      `json:"openapi"`
	Info    openAPIInfo                                                 `json:"info"`
	Paths   *orderedmap.OrderedMap[string, map[string]openAPIOperation] `json:"paths"`
}

type openAPIInfo struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Version     string `json:"version"`
}

type openAPIOperation struct {
	Tags        []string                      `json:"tags,omitempty"`
	Description string                        `json:"description,omitempty"`
	Responses   map[string]openAPIResponse    `json:"responses"`
	Parameters  []jsonschema.OpenAPIParameter `json:"parameters,omitempty"`
	RequestBody *openAPIRequestBody           `json:"requestBody,omitempty"`
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

type OpenAPI struct {
	document        *openAPIObject
	pathMap         *orderedmap.OrderedMap[string, openAPIOperation]
	preHandlerHook  func(ctx.Context, any) ctx.Context
	postHandlerHook func(ctx.Context, any, any)
}

func New(
	title string,
	description string,
	version string,
	codeGen bool,
	codePath string,
	basePkg string,
) *OpenAPI {
	if codeGen {
		jsonschema.InitCommentMap(codePath, basePkg)
	}

	return &OpenAPI{
		document: &openAPIObject{
			OpenAPI: "3.1.0",
			Info: openAPIInfo{
				Title:       title,
				Description: description,
				Version:     version,
			},
			Paths: orderedmap.New[string, map[string]openAPIOperation](),
		},
		pathMap:         orderedmap.New[string, openAPIOperation](),
		preHandlerHook:  nil,
		postHandlerHook: nil,
	}
}

func (o *OpenAPI) RegisterPreHandlerHook(hook func(ctx.Context, any) ctx.Context) {
	o.preHandlerHook = hook
}

func (o *OpenAPI) RegisterPostHandlerHook(hook func(ctx.Context, any, any)) {
	o.postHandlerHook = hook
}

type EchoInterface interface {
	Add(method, path string, handler echo.HandlerFunc, middleware ...echo.MiddlewareFunc) *echo.Route
}

func (o *OpenAPI) GET(
	ech EchoInterface,
	path string,
	handler any,
	options ...PathOption,
) (echo.HandlerFunc, error) {
	return o.Add(ech, http.MethodGet, path, handler, options...)
}

func (o *OpenAPI) POST(
	ech EchoInterface,
	path string,
	handler any,
	options ...PathOption,
) (echo.HandlerFunc, error) {
	return o.Add(ech, http.MethodPost, path, handler, options...)
}

func (o *OpenAPI) PUT(
	ech EchoInterface,
	path string,
	handler any,
	options ...PathOption,
) (echo.HandlerFunc, error) {
	return o.Add(ech, http.MethodPut, path, handler, options...)
}

func (o *OpenAPI) DELETE(
	ech EchoInterface,
	path string,
	handler any,
	options ...PathOption,
) (echo.HandlerFunc, error) {
	return o.Add(ech, http.MethodDelete, path, handler, options...)
}

func (o *OpenAPI) PATCH(
	ech EchoInterface,
	path string,
	handler any,
	options ...PathOption,
) (echo.HandlerFunc, error) {
	return o.Add(ech, http.MethodPatch, path, handler, options...)
}

func (o *OpenAPI) Add(
	ech EchoInterface,
	method string,
	path string,
	handler any,
	options ...PathOption,
) (echo.HandlerFunc, error) {
	description := ""
	middlewares := []echo.MiddlewareFunc{}
	tags := []string{}
	requestContentType := echo.MIMEApplicationJSON
	responseContentType := echo.MIMEApplicationJSON

	for _, option := range options {
		switch opt := option.(type) {
		case Middleware:
			middlewares = append(middlewares, echo.MiddlewareFunc(opt))
		case withDescription:
			description = opt.description
		case withTags:
			tags = append(tags, opt.tags...)
		case withRequestContentType:
			requestContentType = opt.contentType
		case withResponseContentType:
			responseContentType = opt.contentType
		}
	}

	newHandler, handlerName, err := o.createHandler(
		handler,
		path,
		method,
		description,
		tags,
		requestContentType,
		responseContentType,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create handler: %w", err)
	}

	for i := len(middlewares) - 1; i >= 0; i-- {
		newHandler = middlewares[i](newHandler)
	}

	route := ech.Add(method, path, newHandler)
	route.Name = handlerName

	return newHandler, nil
}

var pathParamRegex = regexp.MustCompile(`:(\w+)`)

var errHandlerNotFound = errors.New("handler not found")

func (o *OpenAPI) Flush(ech *echo.Echo) error {
	routes := ech.Routes()

	nameSlice := make([]string, 0, len(routes))
	for _, route := range routes {
		nameSlice = append(nameSlice, route.Name)
	}

	for pair := o.pathMap.Oldest(); pair != nil; pair = pair.Next() {
		nameIdx := slices.Index(nameSlice, pair.Key)
		if nameIdx == -1 {
			return fmt.Errorf("handler %s not found: %w", pair.Key, errHandlerNotFound)
		}

		pathItem := pair.Value
		route := routes[nameIdx]
		firstPath := strings.Split(route.Path, "/")[1]
		pathItem.Tags = append(pathItem.Tags, firstPath)
		method := strings.ToLower(route.Method)

		newPath := pathParamRegex.ReplaceAllString(route.Path, "{$1}")
		if val, ok := o.document.Paths.Get(newPath); ok {
			val[method] = pathItem
			o.document.Paths.Set(newPath, val)
		} else {
			o.document.Paths.Set(newPath, map[string]openAPIOperation{method: pathItem})
		}
	}

	return nil
}

func (o *OpenAPI) Document() ([]byte, error) {
	doc, err := json.MarshalIndent(o.document, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal openapi document: %w", err)
	}

	return doc, nil
}

var errInvalidHandler = errors.New("invalid handler")

func (o *OpenAPI) createHandler(
	handler any,
	path string,
	method string,
	description string,
	tags []string,
	requestContentType string,
	responseContentType string,
) (echo.HandlerFunc, string, error) {
	if inputType, outputType, ok := isRPCHandler(handler); ok {
		handlerName := runtime.FuncForPC(reflect.ValueOf(handler).Pointer()).Name()
		handlerName = fmt.Sprintf("%s.%s.%s", path, method, handlerName)

		handlerFn, err := createRPCHandler(
			handler,
			method,
			inputType,
			requestContentType,
			responseContentType,
			o.preHandlerHook,
			o.postHandlerHook,
		)
		if err != nil {
			return nil, "", fmt.Errorf("failed to create rpc handler: %w", err)
		}

		item, err := getPathItem(inputType, outputType, method, description, tags, requestContentType, responseContentType)
		if err != nil {
			return nil, "", fmt.Errorf("failed to generate path item: %w", err)
		}

		o.pathMap.Set(handlerName, *item)

		return handlerFn, handlerName, nil
	} else {
		return nil, "", fmt.Errorf("handler is not a valid handler: %w", errInvalidHandler)
	}
}

//nolint:gochecknoglobals
var hasBodyMethods = []string{
	http.MethodPost,
	http.MethodPut,
	http.MethodPatch,
}

func getPathItem(
	input reflect.Type,
	output reflect.Type,
	method string,
	description string,
	tags []string,
	requestContentType string,
	responseContentType string,
) (*openAPIOperation, error) {
	outSchema, err := jsonschema.AnyToSchema(output)
	if err != nil {
		return nil, fmt.Errorf("failed to convert output to schema: %w", err)
	}

	operation := openAPIOperation{
		Tags:        tags,
		Description: description,
		Responses: map[string]openAPIResponse{"200": {
			Description: outSchema.Description,
			Content:     map[string]openAPIMediaType{responseContentType: {Schema: *outSchema}},
		}},
		Parameters:  nil,
		RequestBody: nil,
	}

	hasBody := slices.Contains(hasBodyMethods, method)

	operation.Parameters, err = jsonschema.ParametersToSchema(input, !hasBody)
	if err != nil {
		return nil, fmt.Errorf("failed to convert input to schema: %w", err)
	}

	if hasBody {
		inputSchema, err := jsonschema.RequestBodyToSchema(input)
		if err != nil {
			return nil, fmt.Errorf("failed to generate schema for input: %w", err)
		}

		operation.RequestBody = &openAPIRequestBody{
			Description: inputSchema.Description,
			Content:     map[string]openAPIMediaType{requestContentType: {Schema: *inputSchema}},
			Required:    true,
		}
	}

	return &operation, nil
}
