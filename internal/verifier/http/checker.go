package httpverifier

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/JulianoL13/app-proxy-engine/internal/common/logs"
	"github.com/JulianoL13/app-proxy-engine/internal/verifier"
)

type Checker struct {
	TargetURL string
	Timeout   time.Duration
	logger    logs.Logger
	realIP    string
}

func NewChecker(target string, timeout time.Duration, logger logs.Logger) *Checker {
	c := &Checker{
		TargetURL: target,
		Timeout:   timeout,
		logger:    logger,
	}
	c.realIP = c.fetchRealIP()
	return c
}

// fetchRealIP gets the real IP without proxy for comparison
func (c *Checker) fetchRealIP() string {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", "https://httpbin.org/ip", nil)
	if err != nil {
		c.logger.Warn("failed to create request for real IP", "error", err)
		return ""
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.logger.Warn("failed to fetch real IP", "error", err)
		return ""
	}
	defer resp.Body.Close()

	var result struct {
		Origin string `json:"origin"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		c.logger.Warn("failed to decode real IP response", "error", err)
		return ""
	}

	c.logger.Info("detected real IP", "ip", result.Origin)
	return result.Origin
}

func (c *Checker) Verify(ctx context.Context, p verifier.Verifiable) verifier.VerifyOutput {
	proxyURL := p.URL()

	transport := &http.Transport{
		Proxy:             http.ProxyURL(proxyURL),
		DisableKeepAlives: true,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   c.Timeout,
	}

	start := time.Now()

	reqCtx, cancel := context.WithTimeout(ctx, c.Timeout)
	defer cancel()

	// Use /headers endpoint to detect proxy headers
	req, err := http.NewRequestWithContext(reqCtx, "GET", "https://httpbin.org/headers", nil)
	if err != nil {
		return verifier.VerifyOutput{Error: err}
	}

	req.Header.Set("User-Agent", "ProxyEngine/1.0")

	resp, err := client.Do(req)
	latency := time.Since(start)

	if err != nil {
		c.logger.Debug("proxy verification failed", "address", p.Address(), "error", err)
		return verifier.VerifyOutput{
			Success: false,
			Latency: latency,
			Error:   err,
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return verifier.VerifyOutput{
			Success: false,
			Latency: latency,
			Error:   fmt.Errorf("bad status code: %d", resp.StatusCode),
		}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return verifier.VerifyOutput{
			Success: false,
			Latency: latency,
			Error:   fmt.Errorf("failed to read response: %w", err),
		}
	}

	anonymity := c.detectAnonymity(body)

	return verifier.VerifyOutput{
		Success:   true,
		Latency:   latency,
		Anonymity: anonymity,
	}
}

// headersResponse represents httpbin.org/headers response
type headersResponse struct {
	Headers map[string]string `json:"headers"`
}

// detectAnonymity analyzes headers to determine proxy anonymity level
func (c *Checker) detectAnonymity(body []byte) string {
	var resp headersResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return "unknown"
	}

	// Check for real IP leak in common proxy headers
	proxyHeaders := []string{
		"X-Forwarded-For",
		"X-Real-Ip",
		"X-Client-Ip",
		"Forwarded",
		"Client-Ip",
		"Via",
	}

	hasProxyHeader := false
	hasIPLeak := false

	for _, header := range proxyHeaders {
		value, exists := resp.Headers[header]
		if !exists {
			continue
		}

		hasProxyHeader = true

		// Check if our real IP is leaked
		if c.realIP != "" && strings.Contains(value, c.realIP) {
			hasIPLeak = true
			break
		}
	}

	// Classification logic:
	// - transparent: IP is leaked
	// - anonymous: proxy headers exist but no IP leak
	// - elite: no proxy headers at all
	if hasIPLeak {
		return "transparent"
	}
	if hasProxyHeader {
		return "anonymous"
	}
	return "elite"
}
