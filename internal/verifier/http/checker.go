package httpverifier

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/JulianoL13/app-proxy-engine/internal/common/logs"
	"github.com/JulianoL13/app-proxy-engine/internal/verifier"
)

type Checker struct {
	TargetURL string
	Timeout   time.Duration
	logger    logs.Logger
}

func NewChecker(target string, timeout time.Duration, logger logs.Logger) *Checker {
	return &Checker{
		TargetURL: target,
		Timeout:   timeout,
		logger:    logger,
	}
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

	req, err := http.NewRequestWithContext(reqCtx, "GET", c.TargetURL, nil)
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
	// TODO: Parse body to check for real IP leak (Anonymity Check)
	return verifier.VerifyOutput{
		Success:   true,
		Latency:   latency,
		Anonymity: verifier.Elite,
	}
}
