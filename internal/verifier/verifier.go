package verifier

import (
	"net/url"
	"time"
)

type AnonymityLevel string

const (
	Elite       AnonymityLevel = "Elite"
	Anonymous   AnonymityLevel = "Anonymous"
	Transparent AnonymityLevel = "Transparent"
)

type Verifiable interface {
	Address() string
	URL() *url.URL
}

type VerifyOutput struct {
	Success   bool
	Latency   time.Duration
	Anonymity AnonymityLevel
	Error     error
}
