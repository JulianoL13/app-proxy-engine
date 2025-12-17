package proxy

import (
	"context"
	"time"
)

type GetProxiesLogger interface {
	Info(msg string, args ...any)
}

type FilterOptions struct {
	Protocol   string
	Anonymity  string
	MaxLatency time.Duration
}

type Reader interface {
	GetAlive(ctx context.Context, cursor float64, limit int, filter FilterOptions) ([]*Proxy, float64, int, error)
}

type GetProxiesInput struct {
	Cursor     float64
	Limit      int
	Protocol   string
	Anonymity  string
	MaxLatency time.Duration
}

type GetProxiesOutput struct {
	Proxies    []*Proxy
	NextCursor float64
	Total      int
}

type GetProxiesUseCase struct {
	reader Reader
	logger GetProxiesLogger
}

func NewGetProxiesUseCase(reader Reader, logger GetProxiesLogger) *GetProxiesUseCase {
	return &GetProxiesUseCase{
		reader: reader,
		logger: logger,
	}
}

func (uc *GetProxiesUseCase) Execute(ctx context.Context, input GetProxiesInput) (GetProxiesOutput, error) {
	filters := FilterOptions{
		Protocol:   input.Protocol,
		Anonymity:  input.Anonymity,
		MaxLatency: input.MaxLatency,
	}

	proxies, nextCursor, total, err := uc.reader.GetAlive(ctx, input.Cursor, input.Limit, filters)
	if err != nil {
		return GetProxiesOutput{}, err
	}

	uc.logger.Info("fetched proxies", "count", len(proxies), "total", total)

	return GetProxiesOutput{
		Proxies:    proxies,
		NextCursor: nextCursor,
		Total:      total,
	}, nil
}
