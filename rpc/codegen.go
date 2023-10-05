package rpc

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/DimmyJing/valise/jsonschema"
	orderedmap "github.com/wk8/go-ordered-map/v2"
)

func createStub(
	path string,
	method string,
	description string,
	req jsonschema.JSONSchema,
	res jsonschema.JSONSchema,
) (string, error) {
	pathSplit := strings.Split(path, "/")
	origPathName := pathSplit[len(pathSplit)-1]
	pathName := strings.ToUpper(string(origPathName[0])) + origPathName[1:]

	inputType, err := jsonschema.JSONSchemaToTS(&req, "export type "+pathName+"Input = ")
	if err != nil {
		return "", fmt.Errorf("failed to convert json schema to ts: %w", err)
	}

	outputType, err := jsonschema.JSONSchemaToTS(&res, "export type "+pathName+"Output = ")
	if err != nil {
		return "", fmt.Errorf("failed to convert json schema to ts: %w", err)
	}

	operation := fmt.Sprintf(
		"%sexport type %s = {\n  input: %sInput,\n  output: %sOutput,\n  method: \"%s\",\n  path: \"%s\",\n}\n",
		jsonschema.FormatComment(description), pathName, pathName, pathName, method, path,
	)

	types := fmt.Sprintf("%s\n\n%s\n\n%s", inputType, outputType, operation)

	return types, nil
}

var errUnsupportedMethod = errors.New("unsupported method")

func processPath(path openAPIPathItem, pathString string) (string, error) { //nolint:funlen,cyclop
	var (
		method    string
		operation openAPIOperation
	)

	switch {
	case path.Get != nil:
		//nolint:goconst
		method = "get"
		operation = path.Get.openAPIOperation
	case path.Post != nil:
		//nolint:goconst
		method = "post"
		operation = path.Post.openAPIOperation
	case path.Put != nil:
		//nolint:goconst
		method = "put"
		operation = path.Put.openAPIOperation
	case path.Delete != nil:
		//nolint:goconst
		method = "delete"
		operation = path.Delete.openAPIOperation
	default:
		return "", errUnsupportedMethod
	}

	operationDescription := operation.Description

	var requestSchema jsonschema.JSONSchema

	//nolint:nestif
	if method == "get" || method == "delete" {
		var params []openAPIParameter
		if method == "get" {
			params = path.Get.Parameters
		} else if method == "delete" {
			params = path.Delete.Parameters
		}

		requestSchema.Type = "object"
		requestSchema.Properties = orderedmap.New[string, *jsonschema.JSONSchema]()
		requestSchema.AdditionalProperties = &jsonschema.JSONSchemaFalse

		for _, param := range params {
			schema := param.Schema
			requestSchema.Properties.Set(param.Name, &schema)

			if param.Description != "" {
				schema.Description = param.Description
			}

			if param.Required {
				requestSchema.Required = append(requestSchema.Required, param.Name)
			}
		}
	} else if method == "post" || method == "put" {
		var body openAPIRequestBody

		if method == "post" {
			body = path.Post.RequestBody
		} else if method == "put" {
			body = path.Put.RequestBody
		}
		requestSchema = body.Content["application/json"].Schema

		if body.Description != "" {
			requestSchema.Description = body.Description
		}
	}

	responseSchema := operation.Responses["200"].Content["application/json"].Schema

	if operation.Responses["200"].Description != "" {
		responseSchema.Description = operation.Responses["200"].Description
	}

	defs, err := createStub(
		pathString,
		method,
		operationDescription,
		requestSchema,
		responseSchema,
	)
	if err != nil {
		return "", fmt.Errorf("failed to create stub: %w", err)
	}

	return defs, nil
}

func (r *Router) CodeGen(path string) error { //nolint:cyclop
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		//nolint:gomnd
		err := os.Mkdir(path, 0o755)
		if err != nil {
			return fmt.Errorf("error creating directory: %w", err)
		}
	}

	if val, err := r.Document(); err == nil {
		//nolint:gosec,gomnd
		err := os.WriteFile(filepath.Join(path, "swagger.json"), val, 0o644)
		if err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}
	} else {
		return err
	}

	doc := r.document
	files := make(map[string][]string)

	for pair := doc.Paths.Oldest(); pair != nil; pair = pair.Next() {
		key, path := pair.Key, pair.Value

		defs, err := processPath(path, key)
		if err != nil {
			return err
		}

		splitPath := strings.Split(key, "/")
		fileName := splitPath[len(splitPath)-2]

		if val, found := files[fileName]; found {
			val = append(val, defs)
			files[fileName] = val
		} else {
			files[fileName] = []string{defs}
		}
	}

	for key, val := range files {
		var builder strings.Builder

		for _, defs := range val {
			builder.WriteString(defs)
			builder.WriteString("\n\n")
		}

		fileContent := []byte(strings.TrimSpace(builder.String()))

		//nolint:gosec,gomnd
		err := os.WriteFile(filepath.Join(path, key+".ts"), fileContent, 0o644)
		if err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}
	}

	return nil
}
