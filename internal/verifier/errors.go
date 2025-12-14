package verifier

import "errors"

// Sentinel errors for verifier operations
var (
	// ErrProxyDead indicates proxy failed to connect or respond
	ErrProxyDead = errors.New("proxy dead")

	// ErrProxyTimeout indicates proxy timed out during verification
	ErrProxyTimeout = errors.New("proxy timeout")
)
