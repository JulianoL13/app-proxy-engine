package scraper

import (
	"context"
	"time"
)

type SchedulerLogger interface {
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
}

type Publisher interface {
	Publish(ctx context.Context, topic string, payload []byte) error
}

type ScrapedProxy interface {
	IP() string
	Port() int
	Protocol() string
	Source() string
}

type ProxyScraper interface {
	Execute(ctx context.Context) ([]ScrapedProxy, []error)
}

type ProxySerializer interface {
	Serialize(p ScrapedProxy) ([]byte, error)
}

const DefaultTopicVerify = "proxies:verify"

type ScheduleScrapingUseCase struct {
	scraper    ProxyScraper
	serializer ProxySerializer
	publisher  Publisher
	interval   time.Duration
	topic      string
	logger     SchedulerLogger
}

func NewScheduleScrapingUseCase(
	scraper ProxyScraper,
	serializer ProxySerializer,
	publisher Publisher,
	interval time.Duration,
	logger SchedulerLogger,
	topic string,
) *ScheduleScrapingUseCase {
	if topic == "" {
		topic = DefaultTopicVerify
	}
	return &ScheduleScrapingUseCase{
		scraper:    scraper,
		serializer: serializer,
		publisher:  publisher,
		interval:   interval,
		topic:      topic,
		logger:     logger,
	}
}

func (uc *ScheduleScrapingUseCase) Execute(ctx context.Context) error {
	uc.logger.Info("starting scheduler", "interval", uc.interval, "topic", uc.topic)

	uc.runCycle(ctx)

	ticker := time.NewTicker(uc.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			uc.logger.Info("scheduler stopped")
			return ctx.Err()
		case <-ticker.C:
			uc.runCycle(ctx)
		}
	}
}

func (uc *ScheduleScrapingUseCase) runCycle(ctx context.Context) {
	uc.logger.Info("starting scrape cycle")

	proxies, errs := uc.scraper.Execute(ctx)
	if len(errs) > 0 {
		uc.logger.Warn("scrape errors", "count", len(errs))
	}

	published := 0
	for _, scraped := range proxies {
		data, err := uc.serializer.Serialize(scraped)
		if err != nil {
			uc.logger.Warn("failed to serialize proxy", "error", err)
			continue
		}

		if err := uc.publisher.Publish(ctx, uc.topic, data); err != nil {
			uc.logger.Warn("failed to publish proxy", "error", err)
			continue
		}
		published++
	}

	uc.logger.Info("scrape cycle complete", "scraped", len(proxies), "published", published)
}
