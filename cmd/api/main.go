package main

import (
	"context"
	"fmt"
	"log"
	logslog "log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"

	"github.com/JulianoL13/app-proxy-engine/internal/common/logs/slog"
	"github.com/JulianoL13/app-proxy-engine/internal/proxy"
	proxyhttp "github.com/JulianoL13/app-proxy-engine/internal/proxy/http"
	proxyredis "github.com/JulianoL13/app-proxy-engine/internal/proxy/redis"
)

type loggerAdapter struct {
	inner slog.Logger
}

func (l *loggerAdapter) Debug(msg string, args ...any) { l.inner.Debug(msg, args...) }
func (l *loggerAdapter) Info(msg string, args ...any)  { l.inner.Info(msg, args...) }
func (l *loggerAdapter) Warn(msg string, args ...any)  { l.inner.Warn(msg, args...) }
func (l *loggerAdapter) Error(msg string, args ...any) { l.inner.Error(msg, args...) }
func (l *loggerAdapter) With(args ...any) proxyhttp.Logger {
	return &loggerAdapter{inner: l.inner.With(args...)}
}

type getProxiesAdapter struct {
	uc *proxy.GetProxiesUseCase
}

func (a *getProxiesAdapter) Execute(ctx context.Context, input proxyhttp.GetProxiesInput) (proxyhttp.GetProxiesOutput, error) {
	out, err := a.uc.Execute(ctx, proxy.GetProxiesInput{
		Cursor:     input.Cursor,
		Limit:      input.Limit,
		Protocol:   input.Protocol,
		Anonymity:  input.Anonymity,
		MaxLatency: input.MaxLatency,
	})
	if err != nil {
		return proxyhttp.GetProxiesOutput{}, err
	}
	return proxyhttp.GetProxiesOutput{
		Proxies:    out.Proxies,
		NextCursor: out.NextCursor,
		Total:      out.Total,
	}, nil
}

type getRandomProxyAdapter struct {
	uc *proxy.GetRandomProxyUseCase
}

func (a *getRandomProxyAdapter) Execute(ctx context.Context, input proxyhttp.GetRandomProxyInput) (*proxy.Proxy, error) {
	return a.uc.Execute(ctx, proxy.GetRandomProxyInput{
		Protocol:   input.Protocol,
		Anonymity:  input.Anonymity,
		MaxLatency: input.MaxLatency,
	})
}

type Config struct {
	APIPort   string
	RedisAddr string
	RedisPass string
	RedisDB   int
	ProxyTTL  time.Duration
	KeyPrefix string
}

func loadConfig() Config {
	_ = godotenv.Load()

	return Config{
		APIPort:   getEnv("API_PORT", "8080"),
		RedisAddr: getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPass: getEnv("REDIS_PASSWORD", ""),
		RedisDB:   getEnvInt("REDIS_DB", 0),
		ProxyTTL:  time.Duration(getEnvInt("PROXY_TTL_MINUTES", 30)) * time.Minute,
		KeyPrefix: getEnv("REDIS_KEY_PREFIX", "v1"),
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

func main() {
	cfg := loadConfig()

	innerLogger := slog.NewJSON(logslog.LevelInfo)
	logger := &loggerAdapter{inner: innerLogger}
	innerLogger.Info("starting proxy-engine API", "port", cfg.APIPort)

	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPass,
		DB:       cfg.RedisDB,
	})
	defer redisClient.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		innerLogger.Error("failed to connect to redis", "error", err)
		os.Exit(1)
	}
	innerLogger.Info("connected to redis", "addr", cfg.RedisAddr)

	repo := proxyredis.NewRepository(redisClient, cfg.KeyPrefix).WithTTL(cfg.ProxyTTL)

	getProxiesUC := proxy.NewGetProxiesUseCase(repo, innerLogger)
	getRandomUC := proxy.NewGetRandomProxyUseCase(repo, innerLogger)

	handler := proxyhttp.NewHandler(
		&getProxiesAdapter{uc: getProxiesUC},
		&getRandomProxyAdapter{uc: getRandomUC},
		logger,
	)
	router := proxyhttp.NewRouter(handler, logger)

	server := &http.Server{
		Addr:         ":" + cfg.APIPort,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info("listening", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("server shutdown error", "error", err)
	}

	fmt.Println("server stopped")
}
