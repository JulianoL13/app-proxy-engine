package httpclient

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/JulianoL13/app-proxy-engine/internal/scraper"
)

const (
	_           = iota
	KB          = 1 << (10 * iota)
	MB          = 1 << (10 * iota)
	maxBodySize = 10 * MB
)

type Logger interface {
	Debug(msg string, args ...any)
}

type Fetcher struct {
	client *http.Client
	logger Logger
}

func New(logger Logger) *Fetcher {
	return &Fetcher{
		client: &http.Client{
			Timeout: 0,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		logger: logger,
	}
}

func (f *Fetcher) FetchAndParse(ctx context.Context, source scraper.Source) ([]*scraper.ScrapeOutput, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", source.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("source %s: create request: %w", source.Name, err)
	}

	req.Header.Set("User-Agent", "ProxyEngine/1.0")

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("source %s: fetch: %w", source.Name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("source %s: status %d: %w", source.Name, resp.StatusCode, scraper.ErrSourceUnavailable)
	}

	limitedReader := io.LimitReader(resp.Body, maxBodySize)

	var proxies []*scraper.ScrapeOutput
	scanner := bufio.NewScanner(limitedReader)
	for scanner.Scan() {
		if ctx.Err() != nil {
			return proxies, ctx.Err()
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		p, err := parseLine(line, source)
		if err != nil {
			f.logger.Debug("parse error", "line", line, "error", err)
			continue
		}
		proxies = append(proxies, p)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("source %s: scan: %w", source.Name, err)
	}

	return proxies, nil
}

func parseLine(line string, source scraper.Source) (*scraper.ScrapeOutput, error) {
	parts := strings.Split(line, ":")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid format")
	}

	ip := strings.TrimSpace(parts[0])
	portStr := strings.TrimSpace(parts[1])

	if net.ParseIP(ip) == nil {
		return nil, fmt.Errorf("invalid ip: %s", ip)
	}

	port, err := strconv.Atoi(portStr)
	if err != nil || port <= 0 || port > 65535 {
		return nil, fmt.Errorf("invalid port")
	}

	if len(parts) == 4 {
		username := strings.TrimSpace(parts[2])
		password := strings.TrimSpace(parts[3])
		return scraper.NewScrapeOutputWithAuth(ip, port, source.Type, source.Name, username, password), nil
	}

	return scraper.NewScrapeOutput(ip, port, source.Type, source.Name), nil
}
