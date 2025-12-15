package scraper_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/JulianoL13/app-proxy-engine/internal/scraper"
	"github.com/stretchr/testify/assert"
)

type mockProxyScraper struct {
	proxies []scraper.ScrapedProxy
	errs    []error
}

func (m *mockProxyScraper) Execute(ctx context.Context) ([]scraper.ScrapedProxy, []error) {
	return m.proxies, m.errs
}

type mockPublisher struct {
	published [][]byte
	err       error
}

func (m *mockPublisher) Publish(ctx context.Context, topic string, payload []byte) error {
	if m.err != nil {
		return m.err
	}
	m.published = append(m.published, payload)
	return nil
}

type mockSerializer struct {
	err error
}

func (m *mockSerializer) Serialize(p scraper.ScrapedProxy) ([]byte, error) {
	if m.err != nil {
		return nil, m.err
	}
	return []byte(p.IP()), nil
}

type stubScrapedProxy struct {
	ip       string
	port     int
	protocol string
	source   string
}

func (s *stubScrapedProxy) IP() string       { return s.ip }
func (s *stubScrapedProxy) Port() int        { return s.port }
func (s *stubScrapedProxy) Protocol() string { return s.protocol }
func (s *stubScrapedProxy) Source() string   { return s.source }

type schedulerTestLogger struct{}

func (l schedulerTestLogger) Info(msg string, args ...any) {}
func (l schedulerTestLogger) Warn(msg string, args ...any) {}

func TestScheduleScrapingUseCase_runCycle(t *testing.T) {
	logger := schedulerTestLogger{}

	t.Run("publishes scraped proxies", func(t *testing.T) {
		proxies := []scraper.ScrapedProxy{
			&stubScrapedProxy{ip: "1.1.1.1", port: 8080, protocol: "http", source: "test"},
			&stubScrapedProxy{ip: "2.2.2.2", port: 3128, protocol: "socks5", source: "test"},
		}

		scraperMock := &mockProxyScraper{proxies: proxies}
		publisher := &mockPublisher{}
		serializer := &mockSerializer{}

		uc := scraper.NewScheduleScrapingUseCase(scraperMock, serializer, publisher, time.Hour, logger)

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		_ = uc.Execute(ctx)

		assert.Len(t, publisher.published, 2)
	})

	t.Run("handles scraper errors gracefully", func(t *testing.T) {
		scraperMock := &mockProxyScraper{
			proxies: []scraper.ScrapedProxy{},
			errs:    []error{errors.New("fetch failed")},
		}
		publisher := &mockPublisher{}
		serializer := &mockSerializer{}

		uc := scraper.NewScheduleScrapingUseCase(scraperMock, serializer, publisher, time.Hour, logger)

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		err := uc.Execute(ctx)

		assert.ErrorIs(t, err, context.DeadlineExceeded)
		assert.Empty(t, publisher.published)
	})

	t.Run("handles publish errors gracefully", func(t *testing.T) {
		proxies := []scraper.ScrapedProxy{
			&stubScrapedProxy{ip: "1.1.1.1", port: 8080, protocol: "http", source: "test"},
		}

		scraperMock := &mockProxyScraper{proxies: proxies}
		publisher := &mockPublisher{err: errors.New("redis unavailable")}
		serializer := &mockSerializer{}

		uc := scraper.NewScheduleScrapingUseCase(scraperMock, serializer, publisher, time.Hour, logger)

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		_ = uc.Execute(ctx)

		assert.Empty(t, publisher.published)
	})

	t.Run("handles serializer errors gracefully", func(t *testing.T) {
		proxies := []scraper.ScrapedProxy{
			&stubScrapedProxy{ip: "1.1.1.1", port: 8080, protocol: "http", source: "test"},
		}

		scraperMock := &mockProxyScraper{proxies: proxies}
		publisher := &mockPublisher{}
		serializer := &mockSerializer{err: errors.New("marshal failed")}

		uc := scraper.NewScheduleScrapingUseCase(scraperMock, serializer, publisher, time.Hour, logger)

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		_ = uc.Execute(ctx)

		assert.Empty(t, publisher.published)
	})
}
