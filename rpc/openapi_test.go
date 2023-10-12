package rpc_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DimmyJing/valise/ctx"
	"github.com/DimmyJing/valise/rpc"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/stretchr/testify/assert"
)

type testInput1 struct {
	Name string
}

type testOutput1 struct {
	Name string
}

func HandlerTest1(inp testInput1, ctx ctx.Context) (testOutput1, error) {
	return testOutput1(inp), nil
}

func TestBasicRoute(t *testing.T) {
	t.Parallel()

	ech := echo.New()
	oapi := rpc.New("test title", "test description", "1.0.0", true, "../", "github.com/DimmyJing/valise")

	handler, err := oapi.Add(ech, http.MethodGet, "/test", HandlerTest1,
		rpc.Middleware(middleware.CORS()),
		rpc.Middleware(rpc.InitMiddleware(nil, nil, nil)),
		rpc.Middleware(rpc.LogMiddleware()),
		rpc.Middleware(rpc.OTelMiddleware([]string{})),
		rpc.Middleware(rpc.RecoverMiddleware()),
	)
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/test?name=jimmy", nil).WithContext(context.Background())
	rec := httptest.NewRecorder()
	echoCtx := ech.NewContext(req, rec)

	err = handler(echoCtx)
	assert.NoError(t, err)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "{\"name\":\"jimmy\"}\n", rec.Body.String())
}
