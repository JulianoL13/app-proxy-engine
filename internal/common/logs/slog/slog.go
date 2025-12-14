package slog

import (
	"log/slog"
	"os"

	"github.com/JulianoL13/app-proxy-engine/internal/common/logs"
)

type Logger struct {
	logger *slog.Logger
}

func New(level slog.Level) *Logger {
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})
	return &Logger{
		logger: slog.New(handler),
	}
}

func NewJSON(level slog.Level) *Logger {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})
	return &Logger{
		logger: slog.New(handler),
	}
}

func (l *Logger) Debug(msg string, args ...any) {
	l.logger.Debug(msg, args...)
}

func (l *Logger) Info(msg string, args ...any) {
	l.logger.Info(msg, args...)
}

func (l *Logger) Warn(msg string, args ...any) {
	l.logger.Warn(msg, args...)
}

func (l *Logger) Error(msg string, args ...any) {
	l.logger.Error(msg, args...)
}

func (l *Logger) With(args ...any) logs.Logger {
	return &Logger{
		logger: l.logger.With(args...),
	}
}

var _ logs.Logger = (*Logger)(nil)
