package scraper_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/JulianoL13/app-proxy-engine/internal/scraper"
	"github.com/JulianoL13/app-proxy-engine/internal/scraper/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestScrapeProxiesUseCase_Execute(t *testing.T) {
	// Setup
	mockFetcher := mocks.NewFetcher(t)
	sources := []scraper.Source{
		{Name: "Source1", URL: "http://source1.com", Type: "http"},
		{Name: "Source2", URL: "http://source2.com", Type: "http"},
	}
	useCase := scraper.NewScrapeProxiesUseCase(mockFetcher, sources)

	proxy1 := &scraper.ScrapedProxy{IP: "1.1.1.1", Port: 8080, Protocol: "http"}
	proxy2 := &scraper.ScrapedProxy{IP: "2.2.2.2", Port: 8080, Protocol: "http"}
	proxyDuplicate := &scraper.ScrapedProxy{IP: "1.1.1.1", Port: 8080, Protocol: "http"} // Same as proxy1

	mockFetcher.On("FetchAndParse", mock.Anything, sources[0]).Return([]*scraper.ScrapedProxy{proxy1}, nil)
	mockFetcher.On("FetchAndParse", mock.Anything, sources[1]).Return([]*scraper.ScrapedProxy{proxy2, proxyDuplicate}, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := useCase.Execute(ctx)

	assert.NoError(t, err)
	assert.Len(t, result, 2, "Should have 2 unique proxies")

	ips := make(map[string]bool)
	for _, p := range result {
		ips[p.IP] = true
	}
	assert.True(t, ips["1.1.1.1"])
	assert.True(t, ips["2.2.2.2"])

	mockFetcher.AssertExpectations(t)
}

func TestScrapeProxiesUseCase_Execute_ErrorHandling(t *testing.T) {
	mockFetcher := mocks.NewFetcher(t)
	sources := []scraper.Source{
		{Name: "Source1", URL: "http://source1.com", Type: "http"},
	}
	useCase := scraper.NewScrapeProxiesUseCase(mockFetcher, sources)

	mockFetcher.On("FetchAndParse", mock.Anything, sources[0]).Return(nil, errors.New("network error"))

	result, err := useCase.Execute(context.Background())

	assert.NoError(t, err)
	assert.Empty(t, result)
}
