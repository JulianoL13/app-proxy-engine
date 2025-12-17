package proxy

import (
	"context"
	"crypto/rand"
	"errors"
	"math/big"
	"time"
)

var ErrNoProxiesAvailable = errors.New("no proxies available")

type GetRandomProxyLogger interface {
	Info(msg string, args ...any)
	Debug(msg string, args ...any)
}

type GetRandomProxyInput struct {
	Protocol   string
	Anonymity  string
	MaxLatency time.Duration
}

type GetRandomProxyUseCase struct {
	reader Reader
	logger GetRandomProxyLogger
}

func NewGetRandomProxyUseCase(reader Reader, logger GetRandomProxyLogger) *GetRandomProxyUseCase {
	return &GetRandomProxyUseCase{
		reader: reader,
		logger: logger,
	}
}

func (uc *GetRandomProxyUseCase) Execute(ctx context.Context, input GetRandomProxyInput) (*Proxy, error) {
	filters := FilterOptions(input)

	proxies, _, _, err := uc.reader.GetAlive(ctx, 0, 0, filters)
	if err != nil {
		return nil, err
	}

	if len(proxies) == 0 {
		return nil, ErrNoProxiesAvailable
	}

	n, err := rand.Int(rand.Reader, big.NewInt(int64(len(proxies))))
	if err != nil {
		return nil, err
	}
	selected := proxies[n.Int64()]
	uc.logger.Debug("selected random proxy", "address", selected.Address())

	return selected, nil
}
