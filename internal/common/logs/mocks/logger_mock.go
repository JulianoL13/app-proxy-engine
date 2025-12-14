package mocks

import "github.com/JulianoL13/app-proxy-engine/internal/common/logs"

type LoggerMock struct{}

func (LoggerMock) Debug(msg string, args ...any) {}
func (LoggerMock) Info(msg string, args ...any)  {}
func (LoggerMock) Warn(msg string, args ...any)  {}
func (LoggerMock) Error(msg string, args ...any) {}
func (LoggerMock) With(args ...any) logs.Logger  { return LoggerMock{} }

var _ logs.Logger = LoggerMock{}
