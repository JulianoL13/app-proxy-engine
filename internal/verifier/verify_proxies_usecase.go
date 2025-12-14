package verifier

import (
	"context"
	"sync"

	"github.com/JulianoL13/app-proxy-engine/internal/common/logs"
)

type ProxyChecker interface {
	Verify(ctx context.Context, p Verifiable) VerifyOutput
}

type VerifyProxiesUseCase struct {
	checker     ProxyChecker
	concurrency int
	logger      logs.Logger
}

func NewVerifyProxiesUseCase(checker ProxyChecker, concurrency int, logger logs.Logger) *VerifyProxiesUseCase {
	return &VerifyProxiesUseCase{
		checker:     checker,
		concurrency: concurrency,
		logger:      logger,
	}
}

func (uc *VerifyProxiesUseCase) Execute(ctx context.Context, proxies []Verifiable) map[string]VerifyOutput {
	uc.logger.Info("starting proxy verification", "count", len(proxies), "concurrency", uc.concurrency)

	jobs := make(chan Verifiable, len(proxies))
	resultsChan := make(chan struct {
		Addr string
		Res  VerifyOutput
	}, len(proxies))

	var wg sync.WaitGroup

	for i := 0; i < uc.concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for p := range jobs {
				select {
				case <-ctx.Done():
					return
				default:
				}

				res := uc.checker.Verify(ctx, p)

				if !res.Success {
					uc.logger.Debug("proxy verification failed", "address", p.Address(), "error", res.Error)
				}

				resultsChan <- struct {
					Addr string
					Res  VerifyOutput
				}{
					Addr: p.Address(),
					Res:  res,
				}
			}
		}()
	}

	for _, p := range proxies {
		jobs <- p
	}
	close(jobs)

	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	output := make(map[string]VerifyOutput)
	for item := range resultsChan {
		output[item.Addr] = item.Res
	}

	uc.logger.Info("verification completed", "total", len(output))
	return output
}
