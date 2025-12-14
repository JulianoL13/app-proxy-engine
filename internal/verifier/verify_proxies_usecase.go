package verifier

import (
	"context"
	"sync"

	"github.com/JulianoL13/app-proxy-engine/internal/common/logs"
)

type ProxyChecker interface {
	Verify(ctx context.Context, p Verifiable) VerifyOutput
}

// TaskExecutor defines the contract for running tasks concurrently
type TaskExecutor interface {
	Start()
	Submit(job func(ctx context.Context))
	Stop()
}

type VerifyProxiesUseCase struct {
	checker ProxyChecker
	pool    TaskExecutor
	logger  logs.Logger
}

func NewVerifyProxiesUseCase(checker ProxyChecker, pool TaskExecutor, logger logs.Logger) *VerifyProxiesUseCase {
	return &VerifyProxiesUseCase{
		checker: checker,
		pool:    pool,
		logger:  logger,
	}
}

type StreamResult struct {
	Address string
	Output  VerifyOutput
}

// Execute streams results utilizing the injected WorkerPool
func (uc *VerifyProxiesUseCase) Execute(ctx context.Context, proxies []Verifiable) <-chan StreamResult {
	results := make(chan StreamResult, len(proxies))

	go func() {
		defer close(results)

		uc.logger.Info("starting proxy verification", "count", len(proxies))

		// Local WaitGroup to track completion of THIS batch of proxies
		var wg sync.WaitGroup
		wg.Add(len(proxies))

		for _, p := range proxies {
			// Capture variable for closure
			proxy := p

			uc.pool.Submit(func(ctx context.Context) {
				defer wg.Done()

				select {
				case <-ctx.Done():
					return
				default:
				}

				res := uc.checker.Verify(ctx, proxy)

				select {
				case <-ctx.Done():
					return
				case results <- StreamResult{Address: proxy.Address(), Output: res}:
				}
			})
		}

		// Wait for all jobs submitted in this execution to finish
		wg.Wait()
		uc.logger.Info("verification completed")
	}()

	return results
}
