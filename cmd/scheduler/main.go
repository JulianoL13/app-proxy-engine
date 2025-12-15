package main

import (
	"context"
	"encoding/json"
	logslog "log/slog"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/JulianoL13/app-proxy-engine/internal/common/events"
	"github.com/JulianoL13/app-proxy-engine/internal/common/logs/slog"
	queueredis "github.com/JulianoL13/app-proxy-engine/internal/common/queue/redis"
	"github.com/JulianoL13/app-proxy-engine/internal/scraper"
	httpclient "github.com/JulianoL13/app-proxy-engine/internal/scraper/http"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

type scraperAdapter struct {
	uc *scraper.ScrapeProxiesUseCase
}

func (a *scraperAdapter) Execute(ctx context.Context) ([]scraper.ScrapedProxy, []error) {
	results, errs := a.uc.Execute(ctx)
	proxies := make([]scraper.ScrapedProxy, len(results))
	for i, r := range results {
		proxies[i] = r
	}
	return proxies, errs
}

type proxySerializer struct{}

func (s proxySerializer) Serialize(p scraper.ScrapedProxy) ([]byte, error) {
	event := events.ProxyDiscoveredEvent{
		IP:       p.IP(),
		Port:     p.Port(),
		Protocol: p.Protocol(),
		Source:   p.Source(),
	}
	return json.Marshal(event)
}

func main() {
	_ = godotenv.Load()

	redisAddr := getEnv("REDIS_ADDR", "localhost:6379")
	redisPass := getEnv("REDIS_PASSWORD", "")
	redisDB := getEnvInt("REDIS_DB", 0)
	scrapeInterval := time.Duration(getEnvInt("SCRAPE_INTERVAL_MINUTES", 30)) * time.Minute
	redisTopic := getEnv("REDIS_TOPIC_VERIFY", "proxies:verify")

	logger := slog.NewJSON(logslog.LevelInfo)

	redisClient := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: redisPass,
		DB:       redisDB,
	})
	defer redisClient.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		logger.Error("failed to connect to redis", "error", err)
		os.Exit(1)
	}

	publisher := queueredis.NewStreamsClient(redisClient)
	fetcher := httpclient.New(logger)
	sources := scraper.PublicSources()
	scrapeUC := scraper.NewScrapeProxiesUseCase(fetcher, sources, logger)

	scraperAdapt := &scraperAdapter{uc: scrapeUC}
	serializer := proxySerializer{}

	uc := scraper.NewScheduleScrapingUseCase(scraperAdapt, serializer, publisher, scrapeInterval, logger, redisTopic)

	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit
		logger.Info("shutting down...")
		cancel()
	}()

	if err := uc.Execute(ctx); err != nil && err != context.Canceled {
		logger.Error("scheduler error", "error", err)
		os.Exit(1)
	}
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return fallback
}
