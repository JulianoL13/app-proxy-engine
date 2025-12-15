package verifier

import "errors"

var (
	ErrProxyDead    = errors.New("proxy dead")
	ErrProxyTimeout = errors.New("proxy timeout")
)
