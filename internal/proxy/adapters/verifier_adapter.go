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

func (a *VerifierAdapter) Check(ctx context.Context, proxies []*proxy.Proxy) map[string]proxy.CheckOutput {
	verifiables := make([]verifier.Verifiable, len(proxies))
	for i, p := range proxies {
		verifiables[i] = p
	}

	results := a.usecase.Execute(ctx, verifiables)
	output := make(map[string]proxy.CheckOutput)
	for addr, r := range results {
		output[addr] = proxy.CheckOutput{
			Success: r.Success,
			Latency: r.Latency.Milliseconds(),
			Error:   r.Error,
		}
	}

	return output
}
