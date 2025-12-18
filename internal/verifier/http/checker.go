package httpverifier

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/JulianoL13/app-proxy-engine/internal/verifier"
)

type Logger interface {
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Debug(msg string, args ...any)
}

const DefaultVerifyURL = "https://httpbin.org/get"

// Security: campos esperados no response do httpbin
var expectedFields = map[string]bool{
	"args": true, "headers": true, "origin": true, "url": true,
}

// Security: limite de tamanho do payload (típico ~300 bytes, 2KB é generoso)
const maxPayloadSize = 2048

type Checker struct {
	TargetURL    string
	Timeout      time.Duration
	logger       Logger
	realIP       string
	initOnce     sync.Once
	baseline     []byte
	baselineHash string
}

func NewChecker(target string, timeout time.Duration, logger Logger) *Checker {
	if target == "" {
		target = DefaultVerifyURL
	}
	return &Checker{
		TargetURL: target,
		Timeout:   timeout,
		logger:    logger,
	}
}

func (c *Checker) ensureRealIP() {
	c.initOnce.Do(func() {
		c.realIP = c.fetchRealIP()
	})
}

func (c *Checker) fetchRealIP() string {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", c.TargetURL, nil)
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

// Security: busca baseline (payload esperado via request direto)
func (c *Checker) ensureBaseline() {
	if c.baseline != nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", c.TargetURL, nil)
	if err != nil {
		c.logger.Warn("failed to create baseline request", "error", err)
		return
	}
	req.Header.Set("User-Agent", "ProxyEngine/1.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.logger.Warn("failed to fetch baseline", "error", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.logger.Warn("failed to read baseline", "error", err)
		return
	}

	c.baseline = body
	c.baselineHash = c.hashPayload(body)
	c.logger.Info("baseline cached", "hash", c.baselineHash[:16]+"...")
}

// Security: calcula SHA256 do payload
func (c *Checker) hashPayload(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// Security: verifica se payload foi modificado pelo proxy
func (c *Checker) checkIntegrity(body []byte) bool {
	// 1. Verificar tamanho (típico ~300 bytes, >2KB é suspeito)
	if len(body) > maxPayloadSize {
		c.logger.Warn("payload size exceeded limit", "size", len(body), "max", maxPayloadSize)
		return false
	}

	// 2. Verificar campos esperados
	var data map[string]any
	if err := json.Unmarshal(body, &data); err != nil {
		return false // JSON inválido
	}

	for key := range data {
		if !expectedFields[key] {
			c.logger.Warn("unexpected field in response", "field", key)
			return false
		}
	}

	// 3. Verificar headers extras suspeitos (se temos baseline)
	if c.baseline != nil {
		var baselineData map[string]any
		if err := json.Unmarshal(c.baseline, &baselineData); err == nil {
			baseHeaders, _ := baselineData["headers"].(map[string]any)
			proxyHeaders, _ := data["headers"].(map[string]any)

			for key := range proxyHeaders {
				if _, exists := baseHeaders[key]; !exists {
					// Header novo adicionado pelo proxy
					if strings.HasPrefix(strings.ToLower(key), "x-") ||
						strings.Contains(strings.ToLower(key), "inject") ||
						strings.Contains(strings.ToLower(key), "ad") {
						c.logger.Warn("suspicious header injected", "header", key)
						return false
					}
				}
			}
		}
	}

	return true
}

func (c *Checker) Verify(ctx context.Context, p verifier.Verifiable) verifier.VerifyOutput {
	c.ensureRealIP()
	c.ensureBaseline()

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

	req, err := http.NewRequestWithContext(reqCtx, "GET", c.TargetURL, nil)
	if err != nil {
		return verifier.VerifyOutput{Error: err}
	}

	req.Header.Set("User-Agent", "ProxyEngine/1.0")

	resp, err := client.Do(req)
	latency := time.Since(start)

	if err != nil {
		c.logger.Debug("proxy verification failed", "address", p.Address(), "error", err)

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

	var dummy map[string]any
	if err := json.Unmarshal(body, &dummy); err != nil {
		return verifier.VerifyOutput{
			Success: false,
			Latency: latency,
			Error:   fmt.Errorf("proxy returned invalid json: %w", err),
		}
	}

	// Security: verifica integridade do payload
	if !c.checkIntegrity(body) {
		c.logger.Warn("proxy failed integrity check", "address", p.Address())
		return verifier.VerifyOutput{
			Success: false,
			Latency: latency,
			Error:   fmt.Errorf("proxy %s: %w", p.Address(), verifier.ErrPayloadModified),
		}
	}

	anonymity := c.detectAnonymity(body)

	return verifier.VerifyOutput{
		Success:   true,
		Latency:   latency,
		Anonymity: anonymity,
	}
}

type httpbinResponse struct {
	Headers map[string]string `json:"headers"`
	Origin  string            `json:"origin"`
}

func (c *Checker) detectAnonymity(body []byte) string {
	var resp httpbinResponse
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
