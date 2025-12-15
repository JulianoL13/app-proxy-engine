package proxy

import (
	"context"
	"errors"
	"math/rand"
)

var ErrNoProxiesAvailable = errors.New("no proxies available")

type GetRandomProxyLogger interface {
	Info(msg string, args ...any)
	Debug(msg string, args ...any)
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

func (uc *GetRandomProxyUseCase) Execute(ctx context.Context) (*Proxy, error) {
	proxies, _, _, err := uc.reader.GetAlive(ctx, 0, 0, FilterOptions{})
	if err != nil {
		return nil, err
	}

	if len(proxies) == 0 {
		return nil, ErrNoProxiesAvailable
	}

	selected := proxies[rand.Intn(len(proxies))]
	uc.logger.Debug("selected random proxy", "address", selected.Address())

	return selected, nil
}
