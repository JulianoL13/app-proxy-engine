package verifier

import (
	"context"
	"sync"
)

type ProxyChecker interface {
	Verify(ctx context.Context, p Verifiable) Result
}

type VerifyProxiesUseCase struct {
	checker     ProxyChecker
	concurrency int
}

func NewVerifyProxiesUseCase(checker ProxyChecker, concurrency int) *VerifyProxiesUseCase {
	return &VerifyProxiesUseCase{
		checker:     checker,
		concurrency: concurrency,
	}
}

func (uc *VerifyProxiesUseCase) Execute(ctx context.Context, proxies []Verifiable) map[string]Result {
	jobs := make(chan Verifiable, len(proxies))
	resultsChan := make(chan struct {
		Addr string
		Res  Result
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

				resultsChan <- struct {
					Addr string
					Res  Result
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

	output := make(map[string]Result)
	for item := range resultsChan {
		output[item.Addr] = item.Res
	}

	return output
}
