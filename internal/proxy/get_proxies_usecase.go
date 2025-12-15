package proxy

import (
	"context"
)

type GetProxiesLogger interface {
	Info(msg string, args ...any)
}

type Reader interface {
	GetAlive(ctx context.Context, cursor float64, limit int) ([]*Proxy, float64, int, error)
}

type GetProxiesInput struct {
	Cursor float64
	Limit  int
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
	proxies, nextCursor, total, err := uc.reader.GetAlive(ctx, input.Cursor, input.Limit)
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
