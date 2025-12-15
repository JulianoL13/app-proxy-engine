package http

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

type ctxKey string

const loggerKey ctxKey = "logger"

func LoggerMiddleware(logger Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := middleware.GetReqID(r.Context())

			contextLogger := logger.With("request_id", requestID)

			ctx := context.WithValue(r.Context(), loggerKey, contextLogger)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RequestLoggerMiddleware(logger Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			defer func() {
				requestID := middleware.GetReqID(r.Context())
				clientIP := r.RemoteAddr
				if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
					clientIP = realIP
				} else if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
					clientIP = forwarded
				}

				logger.Info("request",
					"request_id", requestID,
					"method", r.Method,
					"path", r.URL.Path,
					"status", ww.Status(),
					"duration_ms", time.Since(start).Milliseconds(),
					"client_ip", clientIP,
					"user_agent", r.UserAgent(),
				)
			}()

			next.ServeHTTP(ww, r)
		})
	}
}

func LoggerFromContext(ctx context.Context) Logger {
	if l, ok := ctx.Value(loggerKey).(Logger); ok {
		return l
	}
	return nil
}
