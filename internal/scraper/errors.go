package scraper

import "errors"

// Sentinel errors for scraper operations
var (
	// ErrSourceUnavailable indicates the source returned a non-OK status (retry may help)
	ErrSourceUnavailable = errors.New("source unavailable")

	// ErrInvalidProxy indicates a proxy line could not be parsed (don't retry)
	ErrInvalidProxy = errors.New("invalid proxy format")
)
