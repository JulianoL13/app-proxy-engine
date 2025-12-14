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

	"github.com/JulianoL13/app-proxy-engine/internal/common/logs/slog"
	proxyhttp "github.com/JulianoL13/app-proxy-engine/internal/proxy/http"
	proxyredis "github.com/JulianoL13/app-proxy-engine/internal/proxy/redis"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

type Config struct {
	APIPort   string
	RedisAddr string
	RedisPass string
	RedisDB   int
	ProxyTTL  time.Duration
}

func loadConfig() Config {
	_ = godotenv.Load()

	cfg := Config{
		APIPort:   getEnv("API_PORT", "8080"),
		RedisAddr: getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPass: getEnv("REDIS_PASSWORD", ""),
		RedisDB:   getEnvInt("REDIS_DB", 0),
		ProxyTTL:  time.Duration(getEnvInt("PROXY_TTL_MINUTES", 30)) * time.Minute,
	}

	return cfg
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

	logger := slog.New(logslog.LevelInfo)
	logger.Info("starting proxy-engine API", "port", cfg.APIPort)

	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPass,
		DB:       cfg.RedisDB,
	})
	defer redisClient.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		logger.Error("failed to connect to redis", "error", err)
		os.Exit(1)
	}
	logger.Info("connected to redis", "addr", cfg.RedisAddr)

	repo := proxyredis.NewRepository(redisClient).WithTTL(cfg.ProxyTTL)

	handler := proxyhttp.NewHandler(repo, logger)
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
