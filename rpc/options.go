package rpc

import "github.com/labstack/echo/v4"

type PathOption interface {
	privatePathOption()
}

type Middleware echo.MiddlewareFunc

func (m Middleware) privatePathOption() {}

type withDescription struct {
	description string
}

func (w withDescription) privatePathOption() {}

func WithDesc(description string) withDescription {
	return withDescription{description: description}
}

type withTags struct {
	tags []string
}

func (w withTags) privatePathOption() {}

func WithTags(tags ...string) withTags {
	return withTags{tags: tags}
}

type withRequestContentType struct {
	contentType string
}

func (w withRequestContentType) privatePathOption() {}

func WithRequestContentType(contentType string) withRequestContentType {
	return withRequestContentType{contentType: contentType}
}

type withResponseContentType struct {
	contentType string
}

func (w withResponseContentType) privatePathOption() {}

func WithResponseContentType(contentType string) withResponseContentType {
	return withResponseContentType{contentType: contentType}
}
