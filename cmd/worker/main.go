package main

import (
	"context"
	"encoding/json"
	logslog "log/slog"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/JulianoL13/app-proxy-engine/internal/common/logs/slog"
	queueredis "github.com/JulianoL13/app-proxy-engine/internal/common/queue/redis"
	"github.com/JulianoL13/app-proxy-engine/internal/proxy"
	proxyredis "github.com/JulianoL13/app-proxy-engine/internal/proxy/redis"
	"github.com/JulianoL13/app-proxy-engine/internal/verifier"
	httpverifier "github.com/JulianoL13/app-proxy-engine/internal/verifier/http"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

type consumerAdapter struct {
	inner *queueredis.StreamsClient
}

func (a *consumerAdapter) Subscribe(ctx context.Context, topic, group, consumer string) (<-chan verifier.Message, error) {
	innerCh, err := a.inner.Subscribe(ctx, topic, group, consumer)
	if err != nil {
		return nil, err
	}

	outCh := make(chan verifier.Message)
	go func() {
		defer close(outCh)
		for msg := range innerCh {
			outCh <- verifier.Message{ID: msg.ID, Payload: msg.Payload}
		}
	}()

	return outCh, nil
}

func (a *consumerAdapter) Ack(ctx context.Context, topic, group, msgID string) error {
	return a.inner.Ack(ctx, topic, group, msgID)
}

type proxyDeserializer struct{}

func (d proxyDeserializer) Deserialize(payload []byte) (verifier.VerifiedProxy, error) {
	var p proxy.Proxy
	if err := json.Unmarshal(payload, &p); err != nil {
		return nil, err
	}
	return &proxyAdapter{inner: &p}, nil
}

type proxyAdapter struct {
	inner *proxy.Proxy
}

func (a *proxyAdapter) Address() string { return a.inner.Address() }
func (a *proxyAdapter) URL() *url.URL   { return a.inner.URL() }
func (a *proxyAdapter) MarkSuccess(latency time.Duration, anonymity string) {
	a.inner.MarkSuccess(latency, proxy.AnonymityLevelFromString(anonymity))
}

type writerAdapter struct {
	inner *proxyredis.Repository
}

func (w *writerAdapter) Save(ctx context.Context, p verifier.VerifiedProxy) error {
	pa := p.(*proxyAdapter)
	return w.inner.Save(ctx, pa.inner)
}

func main() {
	_ = godotenv.Load()

	redisAddr := getEnv("REDIS_ADDR", "localhost:6379")
	redisPass := getEnv("REDIS_PASSWORD", "")
	redisDB := getEnvInt("REDIS_DB", 0)
	verifyTimeout := time.Duration(getEnvInt("VERIFY_TIMEOUT_SECONDS", 10)) * time.Second
	proxyTTL := time.Duration(getEnvInt("PROXY_TTL_MINUTES", 30)) * time.Minute
	consumerName := getEnv("CONSUMER_NAME", mustHostname())

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

	consumer := &consumerAdapter{inner: queueredis.NewStreamsClient(redisClient)}
	checker := httpverifier.NewChecker("", verifyTimeout, logger)
	deserializer := proxyDeserializer{}
	writer := &writerAdapter{inner: proxyredis.NewRepository(redisClient).WithTTL(proxyTTL)}

	uc := verifier.NewVerifyFromQueueUseCase(consumer, checker, deserializer, writer, logger, consumerName)

	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit
		logger.Info("shutting down...")
		cancel()
	}()

	if err := uc.Execute(ctx); err != nil && err != context.Canceled {
		logger.Error("usecase error", "error", err)
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

func mustHostname() string {
	h, _ := os.Hostname()
	return h
}
