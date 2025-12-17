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

type schedulerTestLogger struct{}

func (l schedulerTestLogger) Info(msg string, args ...any) {}
func (l schedulerTestLogger) Warn(msg string, args ...any) {}

func TestScheduleScrapingUseCase_runCycle(t *testing.T) {
	logger := schedulerTestLogger{}

	t.Run("publishes scraped proxies", func(t *testing.T) {
		proxy1 := mocks.NewScrapedProxy(t)
		proxy1.EXPECT().IP().Return("1.1.1.1").Maybe()
		proxy1.EXPECT().Port().Return(8080).Maybe()
		proxy1.EXPECT().Protocol().Return("http").Maybe()
		proxy1.EXPECT().Source().Return("test").Maybe()
		proxy1.EXPECT().Username().Return("").Maybe()
		proxy1.EXPECT().Password().Return("").Maybe()

		proxy2 := mocks.NewScrapedProxy(t)
		proxy2.EXPECT().IP().Return("2.2.2.2").Maybe()
		proxy2.EXPECT().Port().Return(3128).Maybe()
		proxy2.EXPECT().Protocol().Return("socks5").Maybe()
		proxy2.EXPECT().Source().Return("test").Maybe()
		proxy2.EXPECT().Username().Return("").Maybe()
		proxy2.EXPECT().Password().Return("").Maybe()

		scraperMock := mocks.NewProxyScraper(t)
		scraperMock.EXPECT().
			Execute(mock.Anything).
			Return([]scraper.ScrapedProxy{proxy1, proxy2}, []error{})

		publisher := mocks.NewPublisher(t)
		publisher.EXPECT().
			Publish(mock.Anything, "test-topic", mock.Anything).
			Return(nil).Times(2)

		serializer := mocks.NewProxySerializer(t)
		serializer.EXPECT().
			Serialize(mock.Anything).
			Return([]byte("serialized"), nil).Times(2)

		cleaner := mocks.NewCleaner(t)
		cleaner.EXPECT().
			Cleanup(mock.Anything).
			Return(nil)

		uc := scraper.NewScheduleScrapingUseCase(scraperMock, serializer, publisher, cleaner, time.Hour, logger, "test-topic")

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		_ = uc.Execute(ctx)
	})

	t.Run("handles scraper errors gracefully", func(t *testing.T) {
		scraperMock := mocks.NewProxyScraper(t)
		scraperMock.EXPECT().
			Execute(mock.Anything).
			Return([]scraper.ScrapedProxy{}, []error{errors.New("fetch failed")})

		publisher := mocks.NewPublisher(t)
		serializer := mocks.NewProxySerializer(t)

		cleaner := mocks.NewCleaner(t)
		cleaner.EXPECT().
			Cleanup(mock.Anything).
			Return(nil)

		uc := scraper.NewScheduleScrapingUseCase(scraperMock, serializer, publisher, cleaner, time.Hour, logger, "test-topic")

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		err := uc.Execute(ctx)

		assert.ErrorIs(t, err, context.DeadlineExceeded)
	})

	t.Run("handles publish errors gracefully", func(t *testing.T) {
		proxy1 := mocks.NewScrapedProxy(t)
		proxy1.EXPECT().IP().Return("1.1.1.1").Maybe()
		proxy1.EXPECT().Port().Return(8080).Maybe()
		proxy1.EXPECT().Protocol().Return("http").Maybe()
		proxy1.EXPECT().Source().Return("test").Maybe()
		proxy1.EXPECT().Username().Return("").Maybe()
		proxy1.EXPECT().Password().Return("").Maybe()

		scraperMock := mocks.NewProxyScraper(t)
		scraperMock.EXPECT().
			Execute(mock.Anything).
			Return([]scraper.ScrapedProxy{proxy1}, []error{})

		publisher := mocks.NewPublisher(t)
		publisher.EXPECT().
			Publish(mock.Anything, "test-topic", mock.Anything).
			Return(errors.New("redis unavailable"))

		serializer := mocks.NewProxySerializer(t)
		serializer.EXPECT().
			Serialize(mock.Anything).
			Return([]byte("serialized"), nil)

		cleaner := mocks.NewCleaner(t)
		cleaner.EXPECT().
			Cleanup(mock.Anything).
			Return(nil)

		uc := scraper.NewScheduleScrapingUseCase(scraperMock, serializer, publisher, cleaner, time.Hour, logger, "test-topic")

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		_ = uc.Execute(ctx)
	})

	t.Run("handles serializer errors gracefully", func(t *testing.T) {
		proxy1 := mocks.NewScrapedProxy(t)
		proxy1.EXPECT().IP().Return("1.1.1.1").Maybe()
		proxy1.EXPECT().Port().Return(8080).Maybe()
		proxy1.EXPECT().Protocol().Return("http").Maybe()
		proxy1.EXPECT().Source().Return("test").Maybe()
		proxy1.EXPECT().Username().Return("").Maybe()
		proxy1.EXPECT().Password().Return("").Maybe()

		scraperMock := mocks.NewProxyScraper(t)
		scraperMock.EXPECT().
			Execute(mock.Anything).
			Return([]scraper.ScrapedProxy{proxy1}, []error{})

		publisher := mocks.NewPublisher(t)
		serializer := mocks.NewProxySerializer(t)
		serializer.EXPECT().
			Serialize(mock.Anything).
			Return(nil, errors.New("marshal failed"))

		cleaner := mocks.NewCleaner(t)
		cleaner.EXPECT().
			Cleanup(mock.Anything).
			Return(nil)

		uc := scraper.NewScheduleScrapingUseCase(scraperMock, serializer, publisher, cleaner, time.Hour, logger, "test-topic")

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		_ = uc.Execute(ctx)
	})
}
