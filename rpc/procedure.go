package rpc

import (
	"net/http"

	"github.com/DimmyJing/valise/ctx"
	"google.golang.org/protobuf/proto"
)

type Procedure struct {
	middlewares []func(http.Handler) http.Handler
	handler     func(proto.Message, ctx.Context) (proto.Message, error)
	method      string
	tags        []string
	description string
}

func (p *Procedure) With(middleware func(http.Handler) http.Handler) *Procedure {
	newMiddlewares := make([]func(http.Handler) http.Handler, len(p.middlewares)+1)
	copy(newMiddlewares, p.middlewares)
	newMiddlewares[len(p.middlewares)] = middleware

	return &Procedure{
		middlewares: newMiddlewares,
		handler:     p.handler,
		method:      p.method,
		tags:        p.tags,
		description: p.description,
	}
}

func (p *Procedure) Get(handler func(proto.Message, ctx.Context) (proto.Message, error)) *Procedure {
	return &Procedure{
		middlewares: p.middlewares,
		handler:     handler,
		method:      http.MethodGet,
		tags:        p.tags,
		description: p.description,
	}
}

func (p *Procedure) Post(handler func(proto.Message, ctx.Context) (proto.Message, error)) *Procedure {
	return &Procedure{
		middlewares: p.middlewares,
		handler:     handler,
		method:      http.MethodPost,
		tags:        p.tags,
		description: p.description,
	}
}

func (p *Procedure) Put(handler func(proto.Message, ctx.Context) (proto.Message, error)) *Procedure {
	return &Procedure{
		middlewares: p.middlewares,
		handler:     handler,
		method:      http.MethodPut,
		tags:        p.tags,
		description: p.description,
	}
}

func (p *Procedure) Delete(handler func(proto.Message, ctx.Context) (proto.Message, error)) *Procedure {
	return &Procedure{
		middlewares: p.middlewares,
		handler:     handler,
		method:      http.MethodDelete,
		tags:        p.tags,
		description: p.description,
	}
}

func (p *Procedure) WithDesc(description string) *Procedure {
	return &Procedure{
		middlewares: p.middlewares,
		handler:     p.handler,
		method:      p.method,
		tags:        p.tags,
		description: description,
	}
}

func (p *Procedure) WithTags(tags ...string) *Procedure {
	newTags := make([]string, len(p.tags)+len(tags))
	copy(newTags, p.tags)
	copy(newTags[len(p.tags):], tags)

	return &Procedure{
		middlewares: p.middlewares,
		handler:     p.handler,
		method:      p.method,
		tags:        newTags,
		description: p.description,
	}
}
