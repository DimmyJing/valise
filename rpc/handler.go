package rpc

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"reflect"
	"slices"
	"strings"

	"github.com/DimmyJing/valise/jsonschema"
	"github.com/DimmyJing/valise/vctx"
	"github.com/labstack/echo/v4"
)

func isRPCHandler(handler any) (reflect.Type, reflect.Type, bool) {
	handlerFn := reflect.ValueOf(handler)
	handlerFnType := handlerFn.Type()

	if handlerFnType.Kind() != reflect.Func {
		return nil, nil, false
	}

	//nolint:mnd
	if handlerFnType.NumIn() != 2 {
		return nil, nil, false
	}

	//nolint:mnd
	if handlerFnType.NumOut() != 2 {
		return nil, nil, false
	}

	if handlerFnType.Out(0).Kind() != reflect.Struct {
		outType := handlerFnType.Out(0)
		if outType.Kind() != reflect.Slice && outType.Elem().Kind() != reflect.Uint8 {
			return nil, nil, false
		}
	}

	if !handlerFnType.Out(1).Implements(reflect.TypeOf((*error)(nil)).Elem()) {
		return nil, nil, false
	}

	if handlerFnType.In(0).Kind() != reflect.Struct {
		return nil, nil, false
	}

	if handlerFnType.In(1) != reflect.TypeOf(vctx.FromBackground()) {
		return nil, nil, false
	}

	return handlerFnType.In(0), handlerFnType.Out(0), true
}

type inputFieldAttrs struct {
	isList  bool
	isBytes bool
	typ     reflect.Type
	inPath  bool
	inQuery bool
}

var errInvalidTag = errors.New("invalid in tag")

func getInputFieldAttrs(inputType reflect.Type, hasBody bool) (map[string]inputFieldAttrs, error) { //nolint:cyclop
	inputFieldAttrsMap := map[string]inputFieldAttrs{}

	for i := range inputType.NumField() {
		field := inputType.Field(i)
		if !field.IsExported() {
			continue
		}

		fieldAttrs := inputFieldAttrs{typ: field.Type, isList: false, inPath: false, inQuery: !hasBody, isBytes: false}
		fieldName := strings.ToLower(string(field.Name[0])) + field.Name[1:]

		if jsonTag, found := field.Tag.Lookup("json"); found {
			splitTags := strings.Split(jsonTag, ",")
			if len(splitTags) > 0 {
				if splitTags[0] == "-" && len(splitTags) == 1 {
					continue
				} else if splitTags[0] != "" {
					fieldName = splitTags[0]
				}
			}
		}

		if field.Type.Kind() == reflect.Slice || field.Type.Kind() == reflect.Array {
			fieldAttrs.isList = true
		}

		if field.Type.Kind() == reflect.Slice && field.Type.Elem().Kind() == reflect.Uint8 {
			fieldAttrs.isBytes = true
		}

		if inTag, found := field.Tag.Lookup("in"); found {
			switch inTag {
			case "path":
				fieldAttrs.inPath = true
				fieldAttrs.inQuery = false

				if fieldAttrs.isList {
					return nil, fmt.Errorf("cannot use in:path on list field %s: %w", fieldName, errInvalidTag)
				}
			case "query":
				fieldAttrs.inQuery = true
			default:
				return nil, fmt.Errorf("invalid in tag %s: %w", inTag, errInvalidTag)
			}
		}

		inputFieldAttrsMap[fieldName] = fieldAttrs
	}

	return inputFieldAttrsMap, nil
}

func parseInput( //nolint:funlen,gocognit,cyclop
	inputFieldAttrsMap map[string]inputFieldAttrs,
	hasBody bool,
	requestContentType string,
	echoCtx echo.Context,
	ctx vctx.Context,
	inputType reflect.Type,
) (reflect.Value, error) {
	inputValue := reflect.New(inputType).Elem()

	inputMap := make(map[string]any)

	//nolint:nestif
	if hasBody {
		if requestContentType == echo.MIMEApplicationJSON {
			err := json.NewDecoder(echoCtx.Request().Body).Decode(&inputMap)
			if err != nil {
				return inputValue, ctx.Fail(fmt.Errorf("error decoding input json: %w", err))
			}
		} else if requestContentType == echo.MIMEMultipartForm || requestContentType == echo.MIMEApplicationForm {
			for key, value := range inputFieldAttrsMap {
				if value.inQuery || value.inPath {
					continue
				}

				values, err := echoCtx.FormParams()
				if err != nil {
					values = make(map[string][]string)
				}

				switch {
				case !value.isBytes && !value.isList:
					formVal := echoCtx.FormValue(key)
					if formVal != "" {
						inputMap[key] = formVal
					}
				case !value.isBytes:
					if val, ok := values[key]; ok {
						inputMap[key] = val
					}
				default:
					fileHeader, err := echoCtx.FormFile(key)
					if err != nil {
						return inputValue, ctx.Fail(fmt.Errorf("error getting form file %s: %w", key, err))
					}

					file, err := fileHeader.Open()
					if err != nil {
						return inputValue, ctx.Fail(fmt.Errorf("error opening form file %s: %w", key, err))
					}

					res, err := io.ReadAll(file)
					if err != nil {
						return inputValue, ctx.Fail(fmt.Errorf("error reading form file %s: %w", key, err))
					}

					inputMap[key] = res
				}
			}
		}
	}

	values := echoCtx.QueryParams()

	for key, value := range inputFieldAttrsMap {
		//nolint:nestif
		if value.inQuery {
			if value.isList {
				if val, ok := values[key]; ok {
					inputMap[key] = val
				}
			} else {
				inputMap[key] = echoCtx.QueryParam(key)
			}
		} else if value.inPath {
			if param := echoCtx.Param(key); param != "" {
				inputMap[key] = param
			}
		}
	}

	err := jsonschema.AnyToValue(inputMap, inputValue)
	if err != nil {
		return inputValue, ctx.Fail(fmt.Errorf("error converting input %v: %w", inputMap, err))
	}

	return inputValue, nil
}

func createRPCHandler( //nolint:funlen,cyclop,gocognit
	handler any,
	method string,
	inputType reflect.Type,
	requestContentType string,
	responseContentType string,
	preHandlerHook func(vctx.Context, any) vctx.Context,
	postHandlerHook func(vctx.Context, any, any),
) (echo.HandlerFunc, error) {
	hasBody := slices.Contains(hasBodyMethods, method)

	inputFieldAttrsMap, err := getInputFieldAttrs(inputType, hasBody)
	if err != nil {
		return nil, fmt.Errorf("failed to get input field attrs: %w", err)
	}

	handlerValue := reflect.ValueOf(handler)

	return echo.HandlerFunc(func(echoCtx echo.Context) error {
		ctx := FromEchoContext(echoCtx).ctx

		if t, _, err := mime.ParseMediaType(echoCtx.Request().Header.Get("Content-Type")); err == nil {
			requestContentType = t
		}

		inputValue, err := parseInput(
			inputFieldAttrsMap,
			hasBody,
			requestContentType,
			echoCtx,
			ctx,
			inputType,
		)
		if err != nil {
			return ctx.Fail(NewInternalHTTPError(http.StatusBadRequest, err))
		}

		if preHandlerHook != nil {
			ctx = preHandlerHook(ctx, inputValue.Interface())
		}

		out := handlerValue.Call([]reflect.Value{inputValue, reflect.ValueOf(ctx)})
		//nolint:nestif
		if !out[1].IsNil() {
			if err, ok := out[1].Interface().(error); ok {
				var httpError *echo.HTTPError

				if errors.As(err, &httpError) {
					return err
				} else {
					return ctx.Fail(NewInternalHTTPError(http.StatusInternalServerError, err))
				}
			} else {
				//nolint:goerr113
				return ctx.Fail(NewInternalHTTPError(http.StatusInternalServerError,
					fmt.Errorf("non-error value returned from handler: %v", out[1].Interface()),
				))
			}
		}

		output := out[0].Interface()

		if postHandlerHook != nil {
			postHandlerHook(ctx, inputValue.Interface(), output)
		}

		outRes, err := jsonschema.ValueToAny(reflect.ValueOf(output))
		if err != nil {
			return ctx.Fail(NewInternalHTTPError(http.StatusInternalServerError,
				fmt.Errorf("error converting output %v: %w", output, err),
			))
		}

		if responseContentType == echo.MIMEApplicationJSON {
			err := echoCtx.JSON(http.StatusOK, outRes)
			if err != nil {
				return ctx.Fail(NewInternalHTTPError(http.StatusInternalServerError, fmt.Errorf("error writing response: %w", err)))
			}
		} else if bytes, ok := outRes.([]byte); ok {
			err := echoCtx.Blob(http.StatusOK, responseContentType, bytes)
			if err != nil {
				return ctx.Fail(NewInternalHTTPError(http.StatusInternalServerError, fmt.Errorf("error writing response: %w", err)))
			}
		} else {
			return ctx.Fail(NewInternalHTTPError(http.StatusInternalServerError,
				fmt.Errorf("invalid output type %T for content type %s: %w", output, responseContentType, err),
			))
		}

		return nil
	}), nil
}
