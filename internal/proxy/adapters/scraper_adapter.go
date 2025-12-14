package adapters

import (
	"context"

	"github.com/JulianoL13/app-proxy-engine/internal/proxy"
	"github.com/JulianoL13/app-proxy-engine/internal/scraper"
)

type ScraperAdapter struct {
	usecase *scraper.ScrapeProxiesUseCase
}

func NewScraperAdapter(uc *scraper.ScrapeProxiesUseCase) *ScraperAdapter {
	return &ScraperAdapter{usecase: uc}
}

func (a *ScraperAdapter) Fetch(ctx context.Context) ([]proxy.ProxyDataInput, []error) {
	scraped, errs := a.usecase.Execute(ctx)

	result := make([]proxy.ProxyDataInput, len(scraped))
	for i, s := range scraped {
		result[i] = s // ScrapeOutput implements ProxyDataInput
	}

	return result, errs
}
