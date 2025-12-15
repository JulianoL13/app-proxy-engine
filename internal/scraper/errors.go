package scraper

import "errors"

var (
	ErrSourceUnavailable = errors.New("source unavailable")
	ErrInvalidProxy      = errors.New("invalid proxy format")
)
