package workerpool

import (
	"context"
	"fmt"

	"github.com/panjf2000/ants/v2"
)

type Pool struct {
	pool *ants.Pool
}

func New(size int) (*Pool, error) {
	p, err := ants.NewPool(size, ants.WithNonblocking(false))
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}
	return &Pool{pool: p}, nil
}

func (p *Pool) Submit(ctx context.Context, job func(ctx context.Context)) error {
	err := p.pool.Submit(func() {
		if ctx.Err() != nil {
			return
		}
		job(ctx)
	})

	if err != nil {
		return fmt.Errorf("pool submit: %w", err)
	}
	return nil
}

func (p *Pool) Stop() {
	p.pool.Release()
}

func (p *Pool) Workers() int {
	return p.pool.Cap()
}
