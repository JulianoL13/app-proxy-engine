package slog

import (
	"context"
	"log/slog"
	"os"
)

type Logger interface {
	Debug(msg string, args ...any)
	DebugContext(ctx context.Context, msg string, args ...any)
	Info(msg string, args ...any)
	InfoContext(ctx context.Context, msg string, args ...any)
	Warn(msg string, args ...any)
	WarnContext(ctx context.Context, msg string, args ...any)
	Error(msg string, args ...any)
	ErrorContext(ctx context.Context, msg string, args ...any)
	With(args ...any) Logger
}

type logger struct {
	inner *slog.Logger
}

func New(level slog.Level) Logger {
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level:     level,
		AddSource: true,
	})
	return &logger{inner: slog.New(handler)}
}

func NewJSON(level slog.Level) Logger {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     level,
		AddSource: true,
	})
	return &logger{inner: slog.New(handler)}
}

func (l *logger) Debug(msg string, args ...any) {
	l.inner.Debug(msg, args...)
}

func (l *logger) DebugContext(ctx context.Context, msg string, args ...any) {
	l.inner.DebugContext(ctx, msg, args...)
}

func (l *logger) Info(msg string, args ...any) {
	l.inner.Info(msg, args...)
}

func (l *logger) InfoContext(ctx context.Context, msg string, args ...any) {
	l.inner.InfoContext(ctx, msg, args...)
}

func (l *logger) Warn(msg string, args ...any) {
	l.inner.Warn(msg, args...)
}

func (l *logger) WarnContext(ctx context.Context, msg string, args ...any) {
	l.inner.WarnContext(ctx, msg, args...)
}

func (l *logger) Error(msg string, args ...any) {
	l.inner.Error(msg, args...)
}

func (l *logger) ErrorContext(ctx context.Context, msg string, args ...any) {
	l.inner.ErrorContext(ctx, msg, args...)
}

func (l *logger) With(args ...any) Logger {
	return &logger{inner: l.inner.With(args...)}
}
