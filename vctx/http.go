package vctx

import (
	"github.com/labstack/echo/v4"
)

const (
	echoContextKey contextKey = "echoContext"
)

func (c Context) WithEcho(e echo.Context) Context {
	return c.WithValue(echoContextKey, e)
}

func (c Context) Echo() (echo.Context, bool) { //nolint:ireturn
	return Value[echo.Context](c, echoContextKey)
}
