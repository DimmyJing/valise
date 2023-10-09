package ctx

import (
	"net/http"
)

const (
	httpWriterKey  contextKey = "httpResponseWriter"
	httpRequestKey contextKey = "httpRequest"
	httpRouteKey   contextKey = "httpRoute"
)

func FromHTTP(w http.ResponseWriter, r *http.Request) Context {
	return From(r.Context()).WithValue(httpWriterKey, w).WithValue(httpRequestKey, r)
}

func (c Context) GetRequest() (*http.Request, bool) {
	return Value[*http.Request](c, httpRequestKey)
}

func (c Context) GetResponseWriter() (http.ResponseWriter, bool) {
	return Value[http.ResponseWriter](c, httpWriterKey)
}

func (c Context) WithRoute(route string) Context {
	return c.WithValue(httpRouteKey, route)
}

func (c Context) GetRoute() (string, bool) {
	return Value[string](c, httpRouteKey)
}
