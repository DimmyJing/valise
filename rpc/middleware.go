package rpc

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/DimmyJing/valise/attr"
	"github.com/DimmyJing/valise/ctx"
	"github.com/DimmyJing/valise/log"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

type HTTPError struct {
	Code          int
	InternalError error
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("%s: %s", http.StatusText(e.Code), e.InternalError.Error())
}

func (e *HTTPError) Unwrap() error {
	return e.InternalError
}

func NewHTTPError(code int, err error) error {
	return &HTTPError{Code: code, InternalError: err}
}

func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		next.ServeHTTP(w, r)
	})
}

func ErrorMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		defer func() {
			//nolint:nestif
			if rawError := recover(); rawError != nil {
				ctx := ctx.FromHTTP(writer, request)
				err, ok := rawError.(error)
				if !ok {
					//nolint:goerr113
					err = fmt.Errorf("%v", rawError)
				}

				//nolint:errorlint
				if httpErr, ok := err.(*HTTPError); ok {
					writer.WriteHeader(httpErr.Code)
				} else {
					writer.WriteHeader(http.StatusInternalServerError)

					ctx.Error(err.Error(), attr.String("stack", string(debug.Stack())))
				}
				result, jsonError := json.Marshal(struct {
					Error string `json:"error"`
				}{Error: err.Error()})
				if jsonError != nil {
					_, _ = writer.Write([]byte(`{"error": "Internal Server Error"}`))
				} else {
					_, _ = writer.Write(result)
				}

				if ctx.OTelMeter() != nil {
					attributeSet := metric.WithAttributeSet(attribute.NewSet(semconv.HTTPRoute(request.URL.Path)))
					if counter, err := ctx.OTelMeter().Int64Counter("error"); err == nil {
						counter.Add(ctx, 1, attributeSet)
					}
				}
			}
		}()
		next.ServeHTTP(writer, request)
	})
}

func LogMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		ctx := ctx.FromHTTP(writer, request)
		start := time.Now()
		logger := log.Default()
		if ctx.OTelMeter() != nil {
			attributeSet := metric.WithAttributeSet(
				attribute.NewSet(semconv.HTTPRoute(request.URL.Path)),
			)
			if histogram, err := ctx.OTelMeter().Float64Histogram("latency", metric.WithUnit("ms")); err == nil {
				defer func() {
					histogram.Record(
						request.Context(),
						//nolint:gomnd
						float64(time.Since(start).Nanoseconds())/1e6,
						attributeSet,
					)
				}()
			}
			counter, err := ctx.OTelMeter().Int64Counter("request")
			if err == nil {
				counter.Add(request.Context(), 1, attributeSet)
			}
		}

		pathSplit := strings.Split(request.URL.Path, "/")
		spanName := pathSplit[len(pathSplit)-1]
		if ctx.OTelTracer() != nil {
			ctx, rootSpan := ctx.OTelTracer().Start(
				request.Context(),
				spanName,
				trace.WithSpanKind(trace.SpanKindServer),
			)
			request = request.WithContext(ctx)

			defer rootSpan.End()

			logger = logger.With(attr.String("traceID", rootSpan.SpanContext().TraceID().String()))
		}

		logger = logger.With(attr.String("url", request.URL.Path))
		next.ServeHTTP(writer, request)
		logger.Info("called")
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

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		ctx := ctx.FromHTTP(writer, request)
		auth := request.Header.Get("Authorization")
		if auth == "" {
			err := NewHTTPError(http.StatusUnauthorized, errNoAuthHeader)
			log.Panic(ctx.Fail(err))
		}
		parts := strings.Split(auth, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			err := NewHTTPError(
				http.StatusUnauthorized,
				fmt.Errorf("%w: authorization header must be bearer token", errInvalidToken),
			)
			log.Panic(ctx.Fail(err))
		}
		token := parts[1]
		verifier := ctx.TokenVerifier()
		if verifier == nil {
			err := NewHTTPError(http.StatusUnauthorized, errNoTokenVerifier)
			log.Panic(ctx.Fail(err))
		}
		res, err := verifier(ctx, token)
		var userID string
		if err != nil {
			if ctx.IsDevelopment() {
				userID = token
			} else {
				err := NewHTTPError(http.StatusUnauthorized, fmt.Errorf("invalid token: %w", err))
				log.Panic(ctx.Fail(err))
			}
		} else {
			userID = res
		}
		ctx.SetAttributes(attr.String("userID", userID))
		next.ServeHTTP(writer, request.WithContext(ctx.WithUserID(userID)))
	})
}

func MaybeAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		cctx := ctx.FromHTTP(writer, request)
		getUserID := func(ctx ctx.Context) (string, error) {
			auth := request.Header.Get("Authorization")
			if auth == "" {
				return "", NewHTTPError(http.StatusUnauthorized, errNoAuthHeader)
			}
			parts := strings.Split(auth, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				return "", NewHTTPError(
					http.StatusUnauthorized,
					fmt.Errorf("%w: authorization header must be bearer token", errInvalidToken),
				)
			}
			token := parts[1]
			verifier := ctx.TokenVerifier()
			if verifier == nil {
				return "", NewHTTPError(http.StatusUnauthorized, errNoTokenVerifier)
			}
			res, err := verifier(ctx, token)
			var userID string
			if err != nil {
				if ctx.IsDevelopment() {
					userID = token
				} else {
					return "", NewHTTPError(http.StatusUnauthorized, fmt.Errorf("invalid token: %w", err))
				}
			} else {
				userID = res
			}

			return userID, nil
		}
		userID, err := getUserID(cctx)
		if err != nil {
			next.ServeHTTP(writer, request.WithContext(cctx))
		} else {
			cctx.SetAttributes(attr.String("userID", userID))
			next.ServeHTTP(writer, request.WithContext(cctx.WithUserID(userID)))
		}
	})
}
