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

type testLogger struct{}

func (l testLogger) Debug(msg string, args ...any) {}
func (l testLogger) Info(msg string, args ...any)  {}
func (l testLogger) Warn(msg string, args ...any)  {}

func TestScrapeProxiesUseCase_Execute(t *testing.T) {
	ctx := context.Background()
	logger := testLogger{}

	t.Run("success with deduplication", func(t *testing.T) {
		mockFetcher := mocks.NewFetcher(t)

		sources := []scraper.Source{
			{Name: "Source1", URL: "http://source1.com", Type: "http"},
			{Name: "Source2", URL: "http://source2.com", Type: "http"},
		}

		proxy1 := scraper.NewScrapeOutput("1.1.1.1", 8080, "http", "test")
		proxy2 := scraper.NewScrapeOutput("2.2.2.2", 8080, "http", "test")
		proxyDuplicate := scraper.NewScrapeOutput("1.1.1.1", 8080, "http", "test")

		mockFetcher.On("FetchAndParse", mock.Anything, sources[0]).Return([]*scraper.ScrapeOutput{proxy1}, nil)
		mockFetcher.On("FetchAndParse", mock.Anything, sources[1]).Return([]*scraper.ScrapeOutput{proxy2, proxyDuplicate}, nil)

		uc := scraper.NewScrapeProxiesUseCase(mockFetcher, sources, logger)

		timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		result, errs := uc.Execute(timeoutCtx)

		assert.Empty(t, errs)
		assert.Len(t, result, 2, "Should have 2 unique proxies")
		mockFetcher.AssertExpectations(t)
	})

	t.Run("handles fetcher error", func(t *testing.T) {
		mockFetcher := mocks.NewFetcher(t)

		sources := []scraper.Source{
			{Name: "Source1", URL: "http://source1.com", Type: "http"},
		}

		mockFetcher.On("FetchAndParse", mock.Anything, sources[0]).Return(nil, errors.New("network error"))

		uc := scraper.NewScrapeProxiesUseCase(mockFetcher, sources, logger)
		result, errs := uc.Execute(ctx)

		assert.Len(t, errs, 1)
		assert.Empty(t, result)
		mockFetcher.AssertExpectations(t)
	})
}
