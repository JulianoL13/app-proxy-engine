package workerpool

import (
	"context"

	"github.com/panjf2000/ants/v2"
)

type Pool struct {
	pool *ants.Pool
}

func New(size int) *Pool {
	p, _ := ants.NewPool(size, ants.WithNonblocking(false))
	return &Pool{pool: p}
}

func (p *Pool) Start() {}

func (p *Pool) Submit(job func(ctx context.Context)) {
	p.pool.Submit(func() {
		job(context.Background())
	})
}

func (p *Pool) Stop() {
	p.pool.Release()
}

func (p *Pool) Workers() int {
	return p.pool.Cap()
}
