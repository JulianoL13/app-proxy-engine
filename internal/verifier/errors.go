package verifier

import "errors"

var (
	ErrProxyDead         = errors.New("proxy dead")
	ErrProxyTimeout      = errors.New("proxy timeout")
	ErrPayloadModified   = errors.New("proxy modified payload")
	ErrInjectionDetected = errors.New("injection detected in payload")
)
