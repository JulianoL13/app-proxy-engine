package proxy_test

import (
	"context"
	"errors"
	"testing"

	"github.com/JulianoL13/app-proxy-engine/internal/proxy"
	"github.com/JulianoL13/app-proxy-engine/internal/proxy/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type getProxiesTestLogger struct{}

func (l getProxiesTestLogger) Info(msg string, args ...any) {}

func TestGetProxiesUseCase_Execute(t *testing.T) {
	ctx := context.Background()
	logger := getProxiesTestLogger{}

	t.Run("returns proxies successfully", func(t *testing.T) {
		p1 := proxy.NewProxy("1.1.1.1", 8080, proxy.HTTP, "source1")
		p2 := proxy.NewProxy("2.2.2.2", 3128, proxy.SOCKS5, "source2")

		reader := mocks.NewReader(t)
		reader.EXPECT().
			GetAlive(ctx, float64(0), 50, proxy.FilterOptions{}).
			Return([]*proxy.Proxy{p1, p2}, 1234.5, 100, nil)

		uc := proxy.NewGetProxiesUseCase(reader, logger)
		output, err := uc.Execute(ctx, proxy.GetProxiesInput{Cursor: 0, Limit: 50})

		require.NoError(t, err)
		assert.Len(t, output.Proxies, 2)
		assert.Equal(t, 1234.5, output.NextCursor)
		assert.Equal(t, 100, output.Total)
	})

	t.Run("returns empty when no proxies", func(t *testing.T) {
		reader := mocks.NewReader(t)
		reader.EXPECT().
			GetAlive(ctx, float64(0), 50, proxy.FilterOptions{}).
			Return([]*proxy.Proxy{}, float64(0), 0, nil)

		uc := proxy.NewGetProxiesUseCase(reader, logger)
		output, err := uc.Execute(ctx, proxy.GetProxiesInput{Cursor: 0, Limit: 50})

		require.NoError(t, err)
		assert.Empty(t, output.Proxies)
		assert.Equal(t, 0, output.Total)
	})

	t.Run("propagates reader error", func(t *testing.T) {
		reader := mocks.NewReader(t)
		reader.EXPECT().
			GetAlive(ctx, float64(0), 0, proxy.FilterOptions{}).
			Return(nil, float64(0), 0, errors.New("redis connection failed"))

		uc := proxy.NewGetProxiesUseCase(reader, logger)
		_, err := uc.Execute(ctx, proxy.GetProxiesInput{})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "redis connection failed")
	})
}
