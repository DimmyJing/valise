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
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func NewRootRouter(title string, description string, version string, codeGen bool, codePath string) *Router {
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
		jsonschema.InitProtoMap(codePath)
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

func (r *Router) Flush() { //nolint:funlen,gocognit,cyclop
	for _, router := range r.routers {
		path := make([]string, len(r.path)+1)
		copy(path, r.path)
		path[len(r.path)] = router.path
		router.router.path = path
		router.router.document = r.document
		router.router.mux = r.mux
		router.router.Flush()
	}

	for _, proc := range r.procedures {
		path := proc.path
		proc := proc.procedure
		newPath := "/" + strings.Join(r.path, "/") + "/" + path
		method := proc.method

		//nolint:forcetypeassert
		inputMsg := reflect.Zero(reflect.TypeOf(method).In(0)).Interface().(proto.Message)
		inputReflect := inputMsg.ProtoReflect()
		inputFields := inputMsg.ProtoReflect().Descriptor().Fields()
		inputIsList := make(map[string]bool)

		//nolint:forcetypeassert
		outputMsg := reflect.Zero(reflect.TypeOf(method).Out(0)).Interface().(proto.Message)

		for i := 0; i < inputFields.Len(); i++ {
			field := inputFields.Get(i)
			if field.IsList() {
				inputIsList[field.JSONName()] = true
			}
		}

		handlerFn := func(writer http.ResponseWriter, request *http.Request) {
			if request.Method != proc.method {
				log.Panic(NewHTTPError(http.StatusMethodNotAllowed,
					fmt.Errorf("%s not allowed on %s: %w", request.Method, newPath, errMethodNotAllowed)),
				)
			}

			inputMsg := inputReflect.New().Interface()

			//nolint:nestif
			if method == http.MethodGet || method == http.MethodDelete {
				buf := bytes.Buffer{}
				buf.WriteString("{")

				values := request.URL.Query()

				isFirst := true
				for key, value := range values {
					if !isFirst {
						buf.WriteString(fmt.Sprintf(",\"%s\":", key))
					} else {
						isFirst = false
					}

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

				err := protojson.Unmarshal(buf.Bytes(), inputMsg)
				if err != nil {
					log.Panic(NewHTTPError(http.StatusBadRequest, fmt.Errorf("error unmarshaling input: %w", err)))
				}
			} else {
				inputBody, err := io.ReadAll(request.Body)
				if err != nil {
					log.Panic(NewHTTPError(http.StatusBadRequest, fmt.Errorf("error reading body: %w", err)))
				}

				err = protojson.Unmarshal(inputBody, inputMsg)
				if err != nil {
					log.Panic(NewHTTPError(http.StatusBadRequest, fmt.Errorf("error unmarshaling input: %w", err)))
				}
			}

			output, err := proc.handler(inputMsg, ctx.FromHTTP(writer, request))
			if err != nil {
				log.Panic(err)
			}

			if result, err := protojson.Marshal(output); err == nil {
				writer.Header().Set("Content-Type", "application/json")

				_, err = writer.Write(result)
				if err != nil {
					log.Panic(err)
				}
			} else {
				log.Panic(NewHTTPError(http.StatusInternalServerError, fmt.Errorf("error marshaling output: %w", err)))
			}
		}

		handler := http.Handler(http.HandlerFunc(handlerFn))
		for i := len(proc.middlewares) - 1; i >= 0; i-- {
			handler = proc.middlewares[i](handler)
		}

		r.mux.Handle(newPath, handler)

		proc.tags = append(proc.tags, strings.Join(r.path, "/"))

		r.document.addOperation(newPath, inputMsg, outputMsg, method, proc.description, proc.tags)
	}
}
