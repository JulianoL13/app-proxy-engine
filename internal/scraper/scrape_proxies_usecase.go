package scraper

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type Logger interface {
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Debug(msg string, args ...any)
}

type Fetcher interface {
	FetchAndParse(ctx context.Context, source Source) ([]*ScrapeOutput, error)
}

type ScrapeProxiesUseCase struct {
	fetcher       Fetcher
	sources       []Source
	logger        Logger
	sourceTimeout time.Duration
}

func NewScrapeProxiesUseCase(f Fetcher, sources []Source, logger Logger, sourceTimeout time.Duration) *ScrapeProxiesUseCase {
	return &ScrapeProxiesUseCase{
		fetcher:       f,
		sources:       sources,
		logger:        logger,
		sourceTimeout: sourceTimeout,
	}
}

func (uc *ScrapeProxiesUseCase) Execute(ctx context.Context) ([]*ScrapeOutput, []error) {
	uc.logger.Info("starting proxy scrape", "sources", len(uc.sources))

	var wg sync.WaitGroup
	results := make(chan []*ScrapeOutput, len(uc.sources))
	errors := make(chan error, len(uc.sources))

	for _, src := range uc.sources {
		wg.Add(1)
		go func(source Source) {
			defer wg.Done()

			timeoutCtx, cancel := context.WithTimeout(ctx, uc.sourceTimeout)
			defer cancel()

			proxies, err := uc.fetcher.FetchAndParse(timeoutCtx, source)
			if err != nil {
				uc.logger.Warn("source fetch failed", "source", source.Name, "error", err)
				errors <- err
				return
			}
			uc.logger.Debug("source fetched", "source", source.Name, "count", len(proxies))
			results <- proxies
		}(src)
	}

	go func() {
		wg.Wait()
		close(results)
		close(errors)
	}()

	uniqueProxies := make(map[string]*ScrapeOutput)

	var errs []error
	for results != nil || errors != nil {
		select {
		case batch, ok := <-results:
			if !ok {
				results = nil
				continue
			}
			for _, p := range batch {
				key := fmt.Sprintf("%s:%d", p.IP(), p.Port())
				uniqueProxies[key] = p
			}
		case err, ok := <-errors:
			if !ok {
				errors = nil
				continue
			}
			errs = append(errs, err)
		}
	}

	finalList := make([]*ScrapeOutput, 0, len(uniqueProxies))
	for _, p := range uniqueProxies {
		finalList = append(finalList, p)
	}

	uc.logger.Info("scrape completed", "unique", len(uniqueProxies), "errors", len(errs))
	return finalList, errs
}
