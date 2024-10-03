package vctx

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/DimmyJing/valise/log"
)

//nolint:containedctx
type Context struct {
	context.Context
}

type contextKey string

type stringer interface {
	String() string
}

func contextName(c context.Context) string {
	if s, ok := c.(stringer); ok {
		return s.String()
	}

	return reflect.TypeOf(c).String()
}

func From(ctx context.Context) Context {
	return Context{Context: ctx}
}

func FromBackground() Context {
	return Context{Context: context.Background()}
}

func Value[N any](ctx Context, key any) (N, bool) { //nolint:ireturn
	val, ok := ctx.Value(key).(N)

	return val, ok
}

func ValueDefault[N any](ctx Context, key any, def N) N { //nolint:ireturn
	if val, ok := Value[N](ctx, key); ok {
		return val
	}

	return def
}

var errValueNotFound = errors.New("value not found")

func MustValue[N any](ctx Context, key any) N { //nolint:ireturn
	if val, ok := Value[N](ctx, key); ok {
		return val
	}

	log.Panic(fmt.Errorf("%w: %v", errValueNotFound, key))

	return *new(N)
}

func (c Context) String() string {
	return contextName(c.Context) + ".WithValise"
}

func (c Context) WithValue(key any, value any) Context {
	return Context{Context: context.WithValue(c, key, value)}
}

func (c Context) WithDetach() Context {
	return Context{Context: context.WithoutCancel(c)}
}
