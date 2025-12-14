package proxy

import (
	"context"

	"github.com/JulianoL13/app-proxy-engine/internal/common/logs"
)

// Interfaces definidas pelo cliente (este UseCase)

type ProxyDataInput interface {
	IP() string
	Port() int
	Protocol() string
	Source() string
}

type ProxySource interface {
	Fetch(ctx context.Context) ([]ProxyDataInput, []error)
}

type ProxyChecker interface {
	Check(ctx context.Context, proxies []*Proxy) map[string]CheckOutput
}

type CheckOutput struct {
	Success bool
	Latency int64
	Error   error
}

// UseCase

type CollectProxiesUseCase struct {
	source  ProxySource
	checker ProxyChecker
	logger  logs.Logger
}

func NewCollectProxiesUseCase(source ProxySource, checker ProxyChecker, logger logs.Logger) *CollectProxiesUseCase {
	return &CollectProxiesUseCase{
		source:  source,
		checker: checker,
		logger:  logger,
	}
}

func (uc *CollectProxiesUseCase) Execute(ctx context.Context) ([]*Proxy, error) {
	uc.logger.Info("starting proxy collection")

	data, errs := uc.source.Fetch(ctx)
	if len(errs) > 0 {
		uc.logger.Warn("fetch errors", "count", len(errs))
	}
	uc.logger.Info("fetched proxies", "count", len(data))

	if len(data) == 0 {
		return nil, nil
	}

	proxies := make([]*Proxy, len(data))
	for i, d := range data {
		proxies[i] = NewProxy(d.IP(), d.Port(), Protocol(d.Protocol()), d.Source())
	}

	results := uc.checker.Check(ctx, proxies)

	alive := make([]*Proxy, 0)
	for _, p := range proxies {
		result, ok := results[p.Address()]
		if !ok {
			continue
		}
		if result.Success {
			p.MarkSuccess(0, Unknown)
			alive = append(alive, p)
		} else {
			p.MarkFailure()
		}
	}

	uc.logger.Info("collection complete", "alive", len(alive), "dead", len(proxies)-len(alive))
	return alive, nil
}
