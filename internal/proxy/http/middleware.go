package http

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

type ctxKey string

const loggerKey ctxKey = "logger"
const correlationIDKey ctxKey = "correlation_id"

// CorrelationIDMiddleware reads X-Correlation-ID from Kong or falls back to chi's RequestID
func CorrelationIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		correlationID := r.Header.Get("X-Correlation-ID")
		if correlationID == "" {
			correlationID = middleware.GetReqID(r.Context())
		}
		if correlationID == "" {
			correlationID = "unknown"
		}

		ctx := context.WithValue(r.Context(), correlationIDKey, correlationID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetCorrelationID returns the correlation ID from context
func GetCorrelationID(ctx context.Context) string {
	if id, ok := ctx.Value(correlationIDKey).(string); ok {
		return id
	}
	return ""
}

func LoggerMiddleware(logger Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			correlationID := GetCorrelationID(r.Context())

			contextLogger := logger.With("correlation_id", correlationID)

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
				correlationID := GetCorrelationID(r.Context())
				clientIP := r.RemoteAddr
				if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
					clientIP = realIP
				} else if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
					clientIP = forwarded
				}

				logger.Info("request",
					"correlation_id", correlationID,
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
