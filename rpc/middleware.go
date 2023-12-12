package rpc

import (
	"errors"
	"fmt"
	"net/http"
	"runtime/debug"
	"slices"
	"strings"

	"github.com/DimmyJing/valise/attr"
	"github.com/DimmyJing/valise/ctx"
	"github.com/DimmyJing/valise/log"
	"github.com/DimmyJing/valise/otel/otellog"
	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

type Context struct {
	echo.Context
	ctx ctx.Context
}

var _ echo.Context = (*Context)(nil)

func FromEchoContext(echoCtx echo.Context) Context {
	if intEchoCtx, ok := echoCtx.(Context); ok {
		return intEchoCtx
	}

	reqCtx := echoCtx.Request().Context()
	if ctx, ok := reqCtx.(ctx.Context); ok {
		return Context{echoCtx, ctx}
	}

	return Context{echoCtx, ctx.From(reqCtx).WithEcho(echoCtx)}
}

func (c Context) Ctx() ctx.Context {
	return c.ctx
}

func (c Context) WithCtx(ctx ctx.Context) Context {
	c.SetRequest(c.Request().WithContext(ctx))
	c.ctx = ctx

	return c
}

type Handler func(ctx ctx.Context) error

var errHandlerPanic = errors.New("handler panic")

type ErrorMessage struct {
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

func NewInternalHTTPError(code int, err error) *echo.HTTPError {
	return echo.NewHTTPError(code, err).SetInternal(err)
}

func NewHTTPError(code int, messages ...string) *echo.HTTPError {
	switch len(messages) {
	case 0:
		return echo.NewHTTPError(code, ErrorMessage{Message: "", Code: ""})
	case 1:
		return echo.NewHTTPError(code, ErrorMessage{Message: messages[0], Code: ""})
	default:
		return echo.NewHTTPError(code, ErrorMessage{Message: messages[0], Code: messages[1]})
	}
}

func HTTPErrorHandler(err error, echoCtx echo.Context) {
	var httpError *echo.HTTPError

	if errors.As(err, &httpError) {
		if msg, ok := httpError.Message.(ErrorMessage); ok {
			_ = echoCtx.JSON(httpError.Code, msg)
		} else {
			_ = echoCtx.JSON(httpError.Code, ErrorMessage{Code: "", Message: ""})
		}
	} else {
		_ = echoCtx.JSON(http.StatusInternalServerError, ErrorMessage{Code: "", Message: ""})
	}
}

func InitMiddleware(tracer trace.Tracer, meter metric.Meter, logger otellog.Logger) echo.MiddlewareFunc {
	return echo.MiddlewareFunc(func(next echo.HandlerFunc) echo.HandlerFunc {
		return echo.HandlerFunc(func(echoCtx echo.Context) error {
			intEchoCtx := FromEchoContext(echoCtx)
			cctx := intEchoCtx.ctx
			cctx = cctx.WithOTelTracer(tracer)
			cctx = cctx.WithOTelMeter(meter)
			cctx = cctx.WithOTelLog(logger)

			return next(intEchoCtx.WithCtx(cctx))
		})
	})
}

func OTelMiddleware(skipPaths []string) echo.MiddlewareFunc {
	return echo.MiddlewareFunc(func(next echo.HandlerFunc) echo.HandlerFunc {
		return echo.HandlerFunc(func(echoCtx echo.Context) error {
			intEchoCtx := FromEchoContext(echoCtx)
			cctx := intEchoCtx.ctx

			if slices.Contains(skipPaths, echoCtx.Path()) || cctx.OTelTracer() == nil {
				return next(intEchoCtx)
			}

			spanCtx, rootSpan := cctx.OTelTracer().Start(
				cctx,
				echoCtx.Path(),
				trace.WithSpanKind(trace.SpanKindServer),
				trace.WithAttributes(
					semconv.URLFull(echoCtx.Request().URL.String()),
					semconv.URLPath(echoCtx.Request().URL.Path),
					semconv.URLQuery(echoCtx.QueryString()),
					semconv.URLScheme(echoCtx.Scheme()),
					semconv.HTTPRequestMethodKey.String(echoCtx.Request().Method),
					semconv.HTTPRoute(echoCtx.Path()),
					semconv.ServerAddress(echoCtx.Request().Host),
					semconv.UserAgentOriginal(echoCtx.Request().UserAgent()),
					semconv.ClientAddress(echoCtx.RealIP()),
				),
			)
			defer rootSpan.End()

			cctx = ctx.From(spanCtx)

			err := next(intEchoCtx.WithCtx(cctx))

			var httpError *echo.HTTPError

			switch {
			case errors.As(err, &httpError):
				rootSpan.SetAttributes(semconv.HTTPResponseStatusCode(httpError.Code))
			case err != nil:
				rootSpan.SetAttributes(semconv.HTTPResponseStatusCode(http.StatusInternalServerError))
			default:
				rootSpan.SetAttributes(semconv.HTTPResponseStatusCode(http.StatusOK))
			}

			return err
		})
	})
}

func LogMiddleware() echo.MiddlewareFunc {
	return echo.MiddlewareFunc(func(next echo.HandlerFunc) echo.HandlerFunc {
		return echo.HandlerFunc(func(echoCtx echo.Context) error {
			intEchoCtx := FromEchoContext(echoCtx)
			cctx := intEchoCtx.ctx
			logger := log.Default().
				With(attr.String("traceID", trace.SpanFromContext(cctx).SpanContext().TraceID().String())).
				With(attr.String("path", echoCtx.Path()))
			cctx = cctx.WithLog(logger)

			return next(intEchoCtx.WithCtx(cctx))
		})
	})
}

func RecoverMiddleware() echo.MiddlewareFunc {
	return echo.MiddlewareFunc(func(next echo.HandlerFunc) echo.HandlerFunc {
		return echo.HandlerFunc(func(echoCtx echo.Context) error {
			intEchoCtx := FromEchoContext(echoCtx)
			cctx := intEchoCtx.ctx

			defer func() {
				if rawError := recover(); rawError != nil {
					err, ok := rawError.(error)
					if !ok {
						err = fmt.Errorf("%v: %w", rawError, errHandlerPanic)
					}

					stackAttr := attr.String(string(semconv.ExceptionStacktraceKey), string(debug.Stack()))

					cctx.Error(err.Error(), stackAttr)
					// this means that ctx.Fail is not called on the error, so we call it here
					//nolint:errorlint
					if _, ok := err.(*ctx.CallerError); !ok {
						_ = cctx.Fail(err, stackAttr)
					}
				}
			}()

			err := next(intEchoCtx)
			if err != nil {
				cctx.Error(err.Error())

				// this means that ctx.Fail is not called on the error, so we call it here
				//nolint:errorlint
				if _, ok := err.(*ctx.CallerError); !ok {
					_ = cctx.Fail(err)
				}

				return err
			} else {
				cctx.GetLog().Info("ok")
			}

			return nil
		})
	})
}

const UserIDContextKey = "userID"

func UserID(cctx ctx.Context) string {
	return ctx.MustValue[string](cctx, UserIDContextKey)
}

var (
	errNoAuthHeader = fmt.Errorf("no authorization header")
	errInvalidToken = fmt.Errorf("invalid bearer token")
)

type TokenVerifier func(ctx.Context, string) (string, error)

func handleAuth(ctx ctx.Context, auth string, tokenVerifier TokenVerifier, development bool) (string, error) {
	parts := strings.Split(auth, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return "", ctx.Fail(echo.NewHTTPError(
			http.StatusUnauthorized,
			fmt.Errorf("authorization header must be bearer token: %w", errInvalidToken),
		))
	}

	token := parts[1]

	res, err := tokenVerifier(ctx, token)
	if err != nil {
		if development {
			return token, nil
		} else {
			return "", ctx.Fail(echo.NewHTTPError(http.StatusUnauthorized, fmt.Errorf("invalid token: %w", err)))
		}
	} else {
		return res, nil
	}
}

func AuthMiddleware(tokenVerifier TokenVerifier, development bool) echo.MiddlewareFunc {
	return echo.MiddlewareFunc(func(next echo.HandlerFunc) echo.HandlerFunc {
		return echo.HandlerFunc(func(echoCtx echo.Context) error {
			intEchoCtx := FromEchoContext(echoCtx)
			cctx := intEchoCtx.ctx
			request := echoCtx.Request()
			auth := request.Header.Get("Authorization")
			if auth == "" {
				return cctx.Fail(echo.NewHTTPError(http.StatusUnauthorized, errNoAuthHeader))
			}

			userID, err := handleAuth(cctx, auth, tokenVerifier, development)
			if err != nil {
				return err
			}

			cctx.SetAttributes(attr.String(string(semconv.EnduserIDKey), userID))

			cctx = cctx.WithUserID(userID)

			return next(intEchoCtx.WithCtx(cctx))
		})
	})
}

func MaybeAuthMiddleware(tokenVerifier TokenVerifier, development bool) echo.MiddlewareFunc {
	return echo.MiddlewareFunc(func(next echo.HandlerFunc) echo.HandlerFunc {
		return echo.HandlerFunc(func(echoCtx echo.Context) error {
			intEchoCtx := FromEchoContext(echoCtx)
			cctx := intEchoCtx.ctx
			request := echoCtx.Request()
			auth := request.Header.Get("Authorization")
			if auth == "" {
				return next(intEchoCtx)
			}

			userID, err := handleAuth(cctx, auth, tokenVerifier, development)
			if err != nil {
				return err
			}

			cctx.SetAttributes(attr.String(string(semconv.EnduserIDKey), userID))

			cctx = cctx.WithUserID(userID)

			return next(intEchoCtx.WithCtx(cctx))
		})
	})
}
