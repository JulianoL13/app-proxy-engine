package http

import (
	"context"
	"net/http"

	"github.com/JulianoL13/app-proxy-engine/internal/common/logs"
	"github.com/go-chi/chi/v5/middleware"
)

type ctxKey string

const loggerKey ctxKey = "logger"

func LoggerMiddleware(logger logs.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := middleware.GetReqID(r.Context())

			contextLogger := logger.With("request_id", requestID)

			ctx := context.WithValue(r.Context(), loggerKey, contextLogger)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func LoggerFromContext(ctx context.Context) logs.Logger {
	if l, ok := ctx.Value(loggerKey).(logs.Logger); ok {
		return l
	}
	return nil
}
