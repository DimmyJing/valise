package rpc

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/DimmyJing/valise/jsonschema"
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

func (r *Router) Flush() error {
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
		err := registerHandler(proc, r.path, r.mux, r.document)
		if err != nil {
			return fmt.Errorf("failed to register handler: %w", err)
		}
	}

	return nil
}
