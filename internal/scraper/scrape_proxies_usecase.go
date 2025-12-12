package scraper

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type Fetcher interface {
	FetchAndParse(ctx context.Context, source Source) ([]*ScrapedProxy, error)
}

type ScrapeProxiesUseCase struct {
	fetcher Fetcher
	sources []Source
}

func NewScrapeProxiesUseCase(f Fetcher, sources []Source) *ScrapeProxiesUseCase {
	return &ScrapeProxiesUseCase{
		fetcher: f,
		sources: sources,
	}
}

func (uc *ScrapeProxiesUseCase) Execute(ctx context.Context) ([]*ScrapedProxy, error) {
	var wg sync.WaitGroup
	results := make(chan []*ScrapedProxy, len(uc.sources))

	for _, src := range uc.sources {
		wg.Add(1)
		go func(source Source) {
			defer wg.Done()

			timeoutCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
			defer cancel()

			proxies, err := uc.fetcher.FetchAndParse(timeoutCtx, source)
			if err != nil {
				// to-do observability maybe, idk
				return
			}
			results <- proxies
		}(src)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	uniqueProxies := make(map[string]*ScrapedProxy)

	for batch := range results {
		for _, p := range batch {
			uniqueProxies[fmt.Sprintf("%s:%d", p.IP, p.Port)] = p
		}
	}

	finalList := make([]*ScrapedProxy, 0, len(uniqueProxies))
	for _, p := range uniqueProxies {
		finalList = append(finalList, p)
	}

	return finalList, nil
}
