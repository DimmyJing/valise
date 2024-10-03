// TODO: everything

package rpc

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/DimmyJing/valise/jsonschema"
	orderedmap "github.com/wk8/go-ordered-map/v2"
)

func createStub( //nolint:funlen,cyclop
	path string,
	method string,
	description string,
	requestBody *jsonschema.JSONSchema,
	req *jsonschema.JSONSchema,
	res *jsonschema.JSONSchema,
	reqContentType string,
	resContentType string,
) (string, error) {
	pathSplit := strings.Split(path, "/")

	origPathName := "{"
	for splitIdx := len(pathSplit) - 1; strings.Contains(origPathName, "{"); splitIdx-- {
		origPathName = pathSplit[splitIdx]
	}

	pathName := strings.ToUpper(string(origPathName[0])) + origPathName[1:]
	result := ""

	if requestBody != nil {
		inputBodyType, err := jsonschema.JSONSchemaToTS(requestBody, "export type "+pathName+"RequestBody = ")
		if err != nil {
			return "", fmt.Errorf("failed to convert json schema to ts: %w", err)
		}

		result += inputBodyType + "\n\n"
	}

	if req != nil {
		inputType, err := jsonschema.JSONSchemaToTS(req, "export type "+pathName+"Request = ")
		if err != nil {
			return "", fmt.Errorf("failed to convert json schema to ts: %w", err)
		}

		result += inputType + "\n\n"
	}

	outputType, err := jsonschema.JSONSchemaToTS(res, "export type "+pathName+"Response = ")
	if err != nil {
		return "", fmt.Errorf("failed to convert json schema to ts: %w", err)
	}

	result += outputType + "\n\n"

	result += jsonschema.FormatComment(description) + "export type " + pathName + " = {"

	if requestBody != nil {
		result += "\n  body: " + pathName + "RequestBody,"
	}

	if req != nil {
		result += "\n  query: " + pathName + "Request,"
	}

	result += "\n  response: " + pathName + "Response,"
	result += "\n  method: \"" + method + "\","
	result += "\n  path: \"" + path + "\","

	if reqContentType != "" {
		result += "\n  requestContentType: \"" + reqContentType + "\","
	}

	if resContentType != "" {
		result += "\n  responseContentType: \"" + resContentType + "\","
	}

	result += "\n}\n"

	return result, nil
}

var errUnsupportedMethod = errors.New("unsupported method")

func processPath(operation openAPIOperation, method string, pathString string) (string, error) { //nolint:funlen,cyclop
	operationDescription := operation.Description

	var (
		requestBodySchema   *jsonschema.JSONSchema
		requestSchema       *jsonschema.JSONSchema
		requestContentType  string
		responseSchema      *jsonschema.JSONSchema
		responseContentType string
	)

	if slices.Contains(hasBodyMethods, strings.ToUpper(method)) {
		body := operation.RequestBody

		if len(body.Content) != 1 {
			return "", fmt.Errorf("unsupported number of content types %d: %w", len(body.Content), errUnsupportedMethod)
		}

		for key, val := range body.Content {
			schema := val.Schema
			requestContentType = key
			requestBodySchema = &schema
		}

		if body.Description != "" {
			requestBodySchema.Description = body.Description
		}
	}

	params := operation.Parameters
	requestSet := false

	for _, param := range params {
		if param.In != "query" {
			continue
		}

		if !requestSet {
			requestSet = true
			//nolint:exhaustruct
			requestSchema = &jsonschema.JSONSchema{
				Type:                 "object",
				Properties:           orderedmap.New[string, *jsonschema.JSONSchema](),
				AdditionalProperties: &jsonschema.JSONSchemaFalse,
			}
		}

		schema := param.Schema
		requestSchema.Properties.Set(param.Name, schema)

		if param.Description != "" {
			schema.Description = param.Description
		}

		if param.Required {
			requestSchema.Required = append(requestSchema.Required, param.Name)
		}
	}

	if !requestSet && requestBodySchema == nil {
		//nolint:exhaustruct
		requestSchema = &jsonschema.JSONSchema{
			Type:                 "object",
			Properties:           orderedmap.New[string, *jsonschema.JSONSchema](),
			AdditionalProperties: &jsonschema.JSONSchemaFalse,
		}
	}

	if len(operation.Responses["200"].Content) != 1 {
		return "", fmt.Errorf("unsupported number of content types %d: %w",
			len(operation.Responses["200"].Content), errUnsupportedMethod)
	}

	for key, val := range operation.Responses["200"].Content {
		schema := val.Schema
		responseContentType = key
		responseSchema = &schema
	}

	if operation.Responses["200"].Description != "" {
		responseSchema.Description = operation.Responses["200"].Description
	}

	defs, err := createStub(
		pathString,
		method,
		operationDescription,
		requestBodySchema,
		requestSchema,
		responseSchema,
		requestContentType,
		responseContentType,
	)
	if err != nil {
		return "", fmt.Errorf("failed to create stub: %w", err)
	}

	return defs, nil
}

func (o *OpenAPI) CodeGen(path string) error { //nolint:cyclop,funlen
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		//nolint:mnd
		err := os.Mkdir(path, 0o755)
		if err != nil {
			return fmt.Errorf("error creating directory: %w", err)
		}
	}

	if val, err := o.Document(); err == nil {
		//nolint:gosec,mnd
		err := os.WriteFile(filepath.Join(path, "swagger.json"), val, 0o644)
		if err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}
	} else {
		return err
	}

	doc := o.document
	files := make(map[string][]string)

	for pair := doc.Paths.Oldest(); pair != nil; pair = pair.Next() {
		key, path := pair.Key, pair.Value

		for method, pathItem := range path {
			defs, err := processPath(pathItem, method, key)
			if err != nil {
				return err
			}

			splitPath := strings.Split(key, "/")
			fileName := splitPath[1]

			if val, found := files[fileName]; found {
				val = append(val, defs)
				files[fileName] = val
			} else {
				files[fileName] = []string{defs}
			}
		}
	}

	for key, val := range files {
		var builder strings.Builder

		const dateString = "import type { DateString } from \"./common\"\n\n"

		for _, defs := range val {
			builder.WriteString(defs)
			builder.WriteString("\n\n")
		}

		builderString := builder.String()
		if strings.Contains(builderString, "DateString") {
			builderString = dateString + builderString
		}

		fileContent := []byte(strings.TrimSpace(builderString))

		//nolint:gosec,mnd
		err := os.WriteFile(filepath.Join(path, key+".ts"), fileContent, 0o644)
		if err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}
	}

	var builder strings.Builder

	builder.WriteString("export type DateString = string;\n")
	fileContent := []byte(strings.TrimSpace(builder.String()))

	//nolint:gosec,mnd
	err := os.WriteFile(filepath.Join(path, "common.ts"), fileContent, 0o644)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
