package adapters

import (
	"context"

	"github.com/JulianoL13/app-proxy-engine/internal/proxy"
	"github.com/JulianoL13/app-proxy-engine/internal/verifier"
)

type VerifierAdapter struct {
	usecase *verifier.VerifyProxiesUseCase
}

func NewVerifierAdapter(uc *verifier.VerifyProxiesUseCase) *VerifierAdapter {
	return &VerifierAdapter{usecase: uc}
}

func (a *VerifierAdapter) Check(ctx context.Context, proxies []*proxy.Proxy) <-chan proxy.CheckStreamResult {
	results := make(chan proxy.CheckStreamResult)

	verifiables := make([]verifier.Verifiable, len(proxies))
	for i, p := range proxies {
		verifiables[i] = p
	}

	go func() {
		defer close(results)
		for r := range a.usecase.Execute(ctx, verifiables) {
			results <- proxy.CheckStreamResult{
				Address: r.Address,
				Output: proxy.CheckOutput{
					Success: r.Output.Success,
					Latency: r.Output.Latency.Milliseconds(),
					Error:   r.Output.Error,
				},
			}
		}
	}()

	return results
}
