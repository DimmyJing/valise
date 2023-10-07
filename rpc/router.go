package rpc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"

	"github.com/DimmyJing/valise/ctx"
	"github.com/DimmyJing/valise/jsonschema"
	"github.com/DimmyJing/valise/log"
	orderedmap "github.com/wk8/go-ordered-map/v2"
)

func NewRootRouter(
	title string,
	description string,
	version string,
	codeGen bool,
	codePath string,
	basePkg string,
) *Router {
	document := openAPIObject{
		Openapi: "3.1.0",
		Info: openAPIInfo{
			Title:       title,
			Description: description,
			Version:     version,
		},
		Paths: orderedmap.New[string, openAPIPathItem](),
	}

	if codeGen {
		jsonschema.InitCommentMap(codePath, basePkg)
	}

	mux := http.NewServeMux()

	return &Router{
		path:       nil,
		document:   &document,
		mux:        mux,
		procedures: nil,
		routers:    nil,
	}
}

func NewRouter() *Router {
	return &Router{
		path:       nil,
		document:   nil,
		mux:        nil,
		procedures: nil,
		routers:    nil,
	}
}

type routerProcedure struct {
	path      string
	procedure *Procedure
}

type routerRouter struct {
	path   string
	router *Router
}

type Router struct {
	path       []string
	document   *openAPIObject
	mux        *http.ServeMux
	procedures []routerProcedure
	routers    []routerRouter
}

func (r *Router) Procedure(path string, procedure *Procedure) {
	r.procedures = append(r.procedures, routerProcedure{path: path, procedure: procedure})
}

func (r *Router) Route(path string, router *Router) {
	r.routers = append(r.routers, routerRouter{path: path, router: router})
}

func (r *Router) Document() ([]byte, error) {
	doc, err := json.MarshalIndent(r.document, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal openapi document: %w", err)
	}

	return doc, nil
}

func (r *Router) Mux() *http.ServeMux {
	return r.mux
}

var (
	errMethodNotAllowed = fmt.Errorf("method not allowed")
	errBadInput         = fmt.Errorf("bad input")
)

func (r *Router) Flush() error { //nolint:funlen,gocognit,cyclop,gocyclo,maintidx
	for _, router := range r.routers {
		path := make([]string, len(r.path)+1)
		copy(path, r.path)
		path[len(r.path)] = router.path
		router.router.path = path
		router.router.document = r.document
		router.router.mux = r.mux

		err := router.router.Flush()
		if err != nil {
			return fmt.Errorf("failed to flush router: %w", err)
		}
	}

	for _, proc := range r.procedures {
		path := proc.path
		proc := proc.procedure
		newPath := "/" + strings.Join(r.path, "/") + "/" + path
		method := proc.method

		handlerFn := reflect.ValueOf(proc.handler)
		handlerFnType := handlerFn.Type()

		inputMsg := reflect.Zero(handlerFnType.In(0)).Type()
		outputMsg := reflect.Zero(handlerFnType.Out(0)).Type()

		inputIsList := map[string]bool{}

		for i := 0; i < inputMsg.NumField(); i++ {
			field := inputMsg.Field(i)
			if !field.IsExported() {
				continue
			}

			//nolint:nestif
			if field.Type.Kind() == reflect.Slice || field.Type.Kind() == reflect.Array {
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

				inputIsList[fieldName] = true
			}
		}

		httpHandlerFn := func(writer http.ResponseWriter, request *http.Request) {
			if request.Method != proc.method {
				log.Panic(NewHTTPError(http.StatusMethodNotAllowed,
					fmt.Errorf("%s not allowed on %s: %w", request.Method, newPath, errMethodNotAllowed)),
				)
			}

			inputValue := reflect.New(inputMsg).Elem()

			var inputAny any

			//nolint:nestif
			if method == http.MethodGet || method == http.MethodDelete {
				buf := bytes.Buffer{}
				buf.WriteString("{")

				values := request.URL.Query()

				isFirst := true
				for key, value := range values {
					if !isFirst {
						buf.WriteString(",")
					} else {
						isFirst = false
					}

					buf.WriteString(fmt.Sprintf("\"%s\":", key))

					if val, ok := inputIsList[key]; ok && val {
						valBuf, err := json.Marshal(value)
						if err != nil {
							log.Panic(NewHTTPError(http.StatusBadRequest,
								fmt.Errorf("error marshaling input %v for %s: %w", value, key, err)),
							)
						}

						buf.Write(valBuf)
					} else if len(value) == 1 {
						valBuf, err := json.Marshal(value[0])
						if err != nil {
							log.Panic(NewHTTPError(http.StatusBadRequest,
								fmt.Errorf("error marshaling input %v for %s: %w", value, key, err)),
							)
						}
						buf.Write(valBuf)
					} else {
						log.Panic(NewHTTPError(http.StatusBadRequest,
							fmt.Errorf("expect value for %s, but got list %v: %w", key, value, errBadInput)),
						)
					}
				}

				buf.WriteString("}")

				err := json.Unmarshal(buf.Bytes(), &inputAny)
				if err != nil {
					log.Panic(NewHTTPError(http.StatusBadRequest,
						fmt.Errorf("error unmarshaling input %v: %w", buf.String(), err)),
					)
				}
			} else {
				inputBody, err := io.ReadAll(request.Body)
				if err != nil {
					log.Panic(NewHTTPError(http.StatusBadRequest, fmt.Errorf("error reading body: %w", err)))
				}

				err = json.Unmarshal(inputBody, &inputAny)
				if err != nil {
					log.Panic(NewHTTPError(http.StatusBadRequest, fmt.Errorf("error unmarshaling input: %w", err)))
				}
			}

			err := jsonschema.AnyToValue(inputAny, inputValue)
			if err != nil {
				log.Panic(NewHTTPError(http.StatusBadRequest,
					fmt.Errorf("error converting input %v: %w", inputAny, err)),
				)
			}

			out := handlerFn.Call([]reflect.Value{inputValue, reflect.ValueOf(ctx.FromHTTP(writer, request))})
			if !out[1].IsNil() {
				if err, ok := out[1].Interface().(error); ok {
					log.Panic(NewHTTPError(http.StatusInternalServerError, err))
				} else {
					//nolint:goerr113
					log.Panic(NewHTTPError(http.StatusInternalServerError,
						fmt.Errorf("non-error value returned from handler: %v", out[1].Interface()),
					))
				}
			}

			output := out[0].Interface()

			outRes, err := jsonschema.ValueToAny(reflect.ValueOf(output))
			if err != nil {
				log.Panic(NewHTTPError(http.StatusInternalServerError,
					fmt.Errorf("error converting output %v: %w", output, err)),
				)
			}

			if result, err := json.Marshal(outRes); err == nil {
				writer.Header().Set("Content-Type", "application/json")

				if _, err = writer.Write(result); err != nil {
					log.Panic(NewHTTPError(http.StatusInternalServerError, err))
				}
			} else {
				log.Panic(NewHTTPError(http.StatusInternalServerError, fmt.Errorf("error marshaling output: %w", err)))
			}
		}

		handler := http.Handler(http.HandlerFunc(httpHandlerFn))
		for i := len(proc.middlewares) - 1; i >= 0; i-- {
			handler = proc.middlewares[i](handler)
		}

		r.mux.Handle(newPath, handler)

		proc.tags = append(proc.tags, strings.Join(r.path, "/"))

		err := r.document.addOperation(newPath, inputMsg, outputMsg, method, proc.description, proc.tags)
		if err != nil {
			return fmt.Errorf("failed to add operation: %w", err)
		}
	}

	return nil
}
