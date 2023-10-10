package rpc

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"

	"github.com/DimmyJing/valise/attr"
	"github.com/DimmyJing/valise/ctx"
	"github.com/DimmyJing/valise/log"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

type HTTPError struct {
	Code          int
	InternalError error
	ResponseCode  string
	Response      string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("%s: %s", http.StatusText(e.Code), e.InternalError.Error())
}

func (e *HTTPError) Unwrap() error {
	return e.InternalError
}

type Handler func(ctx ctx.Context) error

func NewHTTPError(code int, err error) error {
	return &HTTPError{Code: code, InternalError: err, ResponseCode: "", Response: ""}
}

func applyMiddlewares(handler Handler, middlewares []func(Handler) Handler, route string) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}

	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		err := handler(ctx.FromHTTP(writer, request).WithRoute(route))

		var httpError *HTTPError

		if errors.As(err, &httpError) {
			writer.WriteHeader(httpError.Code)
			res, _ := json.Marshal(struct {
				Code    string `json:"code,omitempty"`
				Message string `json:"message,omitempty"`
			}{Code: httpError.ResponseCode, Message: httpError.Response})
			_, _ = writer.Write(res)
		} else if err != nil {
			writer.WriteHeader(http.StatusInternalServerError)
			_, _ = writer.Write([]byte("{}"))
		}
	})
}

var errHandlerPanic = errors.New("handler panic")

func ErrorMiddleware(next Handler) Handler {
	return Handler(func(cctx ctx.Context) error {
		defer func() {
			if rawError := recover(); rawError != nil {
				err, ok := rawError.(error)
				if !ok {
					err = fmt.Errorf("%v: %w", rawError, errHandlerPanic)
				}

				cctx.Error(err.Error(), attr.String(string(semconv.ExceptionStacktraceKey), string(debug.Stack())))
				// this means that ctx.Fail is not called on the error, so we call it here
				//nolint:errorlint
				if _, ok := err.(*ctx.CallerError); !ok {
					_ = cctx.Fail(err, attr.String(string(semconv.ExceptionStacktraceKey), string(debug.Stack())))
				}
			}
		}()

		err := next(cctx)
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
}

func CORSMiddleware(next Handler) Handler {
	return Handler(func(ctx ctx.Context) error {
		writer, _ := ctx.GetResponseWriter()

		writer.Header().Set("Access-Control-Allow-Origin", "*")
		writer.Header().Set("Access-Control-Allow-Methods", strings.Join([]string{
			http.MethodGet, http.MethodHead, http.MethodPost, http.MethodOptions,
			http.MethodPut, http.MethodPatch, http.MethodDelete,
		}, ", "))
		writer.Header().Set("Access-Control-Allow-Headers", "*")

		request, _ := ctx.GetRequest()
		if request.Method == http.MethodOptions {
			writer.WriteHeader(http.StatusOK)

			return nil
		}

		return next(ctx)
	})
}

func OTelMiddleware(next Handler) Handler {
	return Handler(func(cctx ctx.Context) error {
		if cctx.OTelTracer() == nil {
			return next(cctx)
		}

		tracer := cctx.OTelTracer()
		request, _ := cctx.GetRequest()
		route, _ := cctx.GetRoute()

		var (
			remoteIP   string
			remotePort int
		)

		remoteAddrSplit := strings.Split(request.RemoteAddr, ":")
		//nolint:gomnd
		if len(remoteAddrSplit) == 2 {
			remoteIP = remoteAddrSplit[0]
			if port, err := strconv.Atoi(remoteAddrSplit[1]); err == nil {
				remotePort = port
			}
		}

		spanCtx, rootSpan := tracer.Start(
			cctx,
			route,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				semconv.URLFull(request.URL.String()),
				semconv.URLPath(request.URL.Path),
				semconv.URLQuery(request.URL.RawQuery),
				semconv.URLScheme(request.URL.Scheme),
				semconv.HTTPRequestMethodKey.String(request.Method),
				semconv.HTTPRoute(route),
				semconv.ServerAddress(request.Host),
				semconv.UserAgentOriginal(request.UserAgent()),
				semconv.ClientAddress(remoteIP),
				semconv.ClientPort(remotePort),
			),
		)
		defer rootSpan.End()

		cctx = ctx.From(spanCtx)

		err := next(cctx)

		var httpError *HTTPError

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
}

func LogMiddleware(next Handler) Handler {
	return Handler(func(cctx ctx.Context) error {
		route, _ := cctx.GetRoute()
		logger := log.Default().
			With(attr.String("traceID", trace.SpanFromContext(cctx).SpanContext().TraceID().String())).
			With(attr.String("url", route))
		cctx = cctx.WithLog(logger)

		return next(cctx)
	})
}

const UserIDContextKey = "userID"

func UserID(cctx ctx.Context) string {
	return ctx.MustValue[string](cctx, UserIDContextKey)
}

var (
	errNoAuthHeader    = fmt.Errorf("no authorization header")
	errInvalidToken    = fmt.Errorf("invalid bearer token")
	errNoTokenVerifier = fmt.Errorf("no token verifier")
)

func handleAuth(ctx ctx.Context, auth string) (string, error) {
	parts := strings.Split(auth, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return "", ctx.Fail(NewHTTPError(
			http.StatusUnauthorized,
			fmt.Errorf("authorization header must be bearer token: %w", errInvalidToken),
		))
	}

	token := parts[1]

	verifier := ctx.TokenVerifier()
	if verifier == nil {
		return "", ctx.Fail(NewHTTPError(http.StatusUnauthorized, errNoTokenVerifier))
	}

	res, err := verifier(ctx, token)
	if err != nil {
		if ctx.IsDevelopment() {
			return token, nil
		} else {
			return "", ctx.Fail(NewHTTPError(http.StatusUnauthorized, fmt.Errorf("invalid token: %w", err)))
		}
	} else {
		return res, nil
	}
}

func AuthMiddleware(next Handler) Handler {
	return Handler(func(ctx ctx.Context) error {
		request, _ := ctx.GetRequest()
		auth := request.Header.Get("Authorization")
		if auth == "" {
			return ctx.Fail(NewHTTPError(http.StatusUnauthorized, errNoAuthHeader))
		}

		userID, err := handleAuth(ctx, auth)
		if err != nil {
			return err
		}

		ctx.SetAttributes(attr.String(string(semconv.EnduserIDKey), userID))

		return next(ctx.WithUserID(userID))
	})
}

func MaybeAuthMiddleware(next Handler) Handler {
	return Handler(func(ctx ctx.Context) error {
		request, _ := ctx.GetRequest()
		auth := request.Header.Get("Authorization")
		if auth == "" {
			return next(ctx)
		}

		userID, err := handleAuth(ctx, auth)
		if err != nil {
			return err
		}

		ctx.SetAttributes(attr.String(string(semconv.EnduserIDKey), userID))

		return next(ctx.WithUserID(userID))
	})
}
