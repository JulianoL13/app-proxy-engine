package httpclient

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/JulianoL13/app-proxy-engine/internal/scraper"
)

// Fetcher implementation for HTTP.
type Fetcher struct {
	client *http.Client
}

func New() *Fetcher {
	return &Fetcher{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// FetchAndParse downloads a source and converts it to Proxy structs.
func (f *Fetcher) FetchAndParse(ctx context.Context, source scraper.Source) ([]*scraper.ScrapedProxy, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", source.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("bad request: %w", err)
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status: %d", resp.StatusCode)
	}

	var proxies []*scraper.ScrapedProxy
	scanner := bufio.NewScanner(resp.Body)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		p, err := parseLine(line, source)
		if err == nil {
			proxies = append(proxies, p)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan error: %w", err)
	}

	return proxies, nil
}

func parseLine(line string, source scraper.Source) (*scraper.ScrapedProxy, error) {
	parts := strings.Split(line, ":")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid format")
	}

	ip := parts[0]
	portStr := parts[1]

	if strings.Count(ip, ".") != 3 {
		return nil, fmt.Errorf("invalid ip")
	}

	port, err := strconv.Atoi(portStr)
	if err != nil || port <= 0 || port > 65535 {
		return nil, fmt.Errorf("invalid port")
	}

	return &scraper.ScrapedProxy{
		IP:       ip,
		Port:     port,
		Protocol: source.Type,
		Source:   source.Name,
	}, nil
}
