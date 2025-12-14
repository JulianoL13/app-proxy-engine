package proxy

import (
	"context"

	"github.com/JulianoL13/app-proxy-engine/internal/common/logs"
)

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
	Check(ctx context.Context, proxies []*Proxy) <-chan CheckStreamResult
}

// Writer persists verified proxies
type Writer interface {
	Save(ctx context.Context, p *Proxy) error
}

type CheckOutput struct {
	Success   bool
	Latency   int64
	Anonymity AnonymityLevel
	Error     error
}

type CheckStreamResult struct {
	Address string
	Output  CheckOutput
}

type CollectProxiesUseCase struct {
	source  ProxySource
	checker ProxyChecker
	writer  Writer
	logger  logs.Logger
}

func NewCollectProxiesUseCase(source ProxySource, checker ProxyChecker, writer Writer, logger logs.Logger) *CollectProxiesUseCase {
	return &CollectProxiesUseCase{
		source:  source,
		checker: checker,
		writer:  writer,
		logger:  logger,
	}
}

func (uc *CollectProxiesUseCase) Execute(ctx context.Context) (<-chan *Proxy, error) {
	uc.logger.Info("starting proxy collection")

	data, errs := uc.source.Fetch(ctx)
	if len(errs) > 0 {
		uc.logger.Warn("fetch errors", "count", len(errs))
	}
	uc.logger.Info("fetched proxies", "count", len(data))

	if len(data) == 0 {
		ch := make(chan *Proxy)
		close(ch)
		return ch, nil
	}

	proxies := make([]*Proxy, len(data))
	proxyMap := make(map[string]*Proxy)
	for i, d := range data {
		proxies[i] = NewProxy(d.IP(), d.Port(), Protocol(d.Protocol()), d.Source())
		proxyMap[proxies[i].Address()] = proxies[i]
	}

	results := make(chan *Proxy)

	go func() {
		defer close(results)

		alive := 0
		dead := 0

		for r := range uc.checker.Check(ctx, proxies) {
			p, ok := proxyMap[r.Address]
			if !ok {
				continue
			}

			if r.Output.Success {
				p.MarkSuccess(0, r.Output.Anonymity)
				alive++

				if uc.writer != nil {
					if err := uc.writer.Save(ctx, p); err != nil {
						uc.logger.Warn("failed to save proxy", "address", p.Address(), "error", err)
					}
				}

				results <- p
			} else {
				p.MarkFailure()
				dead++
			}
		}

		uc.logger.Info("collection complete", "alive", alive, "dead", dead)
	}()

	return results, nil
}
