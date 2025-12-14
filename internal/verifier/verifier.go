package verifier

import (
	"net/url"
	"time"
)

type Verifiable interface {
	Address() string
	URL() *url.URL
}

type VerifyOutput struct {
	Success   bool
	Latency   time.Duration
	Anonymity string
	Error     error
}
