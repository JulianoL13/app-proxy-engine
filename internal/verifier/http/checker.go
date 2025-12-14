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

	req, err := http.NewRequestWithContext(reqCtx, "GET", "https://httpbin.org/headers", nil)
	if err != nil {
		return verifier.VerifyOutput{Error: err}
	}

	req.Header.Set("User-Agent", "ProxyEngine/1.0")

	resp, err := client.Do(req)
	latency := time.Since(start)

	if err != nil {
		c.logger.Debug("proxy verification failed", "address", p.Address(), "error", err)

		// Wrap with context and appropriate sentinel
		wrappedErr := fmt.Errorf("proxy %s: %w", p.Address(), err)
		if ctx.Err() == context.DeadlineExceeded || reqCtx.Err() == context.DeadlineExceeded {
			wrappedErr = fmt.Errorf("proxy %s: %w", p.Address(), verifier.ErrProxyTimeout)
		}

		return verifier.VerifyOutput{
			Success: false,
			Latency: latency,
			Error:   wrappedErr,
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return verifier.VerifyOutput{
			Success: false,
			Latency: latency,
			Error:   fmt.Errorf("proxy %s: status %d: %w", p.Address(), resp.StatusCode, verifier.ErrProxyDead),
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

type headersResponse struct {
	Headers map[string]string `json:"headers"`
}

func (c *Checker) detectAnonymity(body []byte) string {
	var resp headersResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return "unknown"
	}

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

		if c.realIP != "" && strings.Contains(value, c.realIP) {
			hasIPLeak = true
			break
		}
	}

	if hasIPLeak {
		return "transparent"
	}
	if hasProxyHeader {
		return "anonymous"
	}
	return "elite"
}
