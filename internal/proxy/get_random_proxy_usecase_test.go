package proxy_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/JulianoL13/app-proxy-engine/internal/proxy"
	"github.com/JulianoL13/app-proxy-engine/internal/proxy/mocks"
)

type getRandomTestLogger struct{}

func (l getRandomTestLogger) Info(msg string, args ...any)  {}
func (l getRandomTestLogger) Debug(msg string, args ...any) {}

func TestGetRandomProxyUseCase_Execute(t *testing.T) {
	ctx := context.Background()
	logger := getRandomTestLogger{}

	t.Run("returns random proxy", func(t *testing.T) {
		p1 := proxy.NewProxy("1.1.1.1", 8080, proxy.HTTP, "source1")
		p1.MarkSuccess(100*time.Millisecond, proxy.Elite)

		reader := mocks.NewReader(t)
		reader.EXPECT().
			GetAlive(ctx, float64(0), 0, mock.AnythingOfType("proxy.FilterOptions")).
			Return([]*proxy.Proxy{p1}, float64(0), 1, nil)

		uc := proxy.NewGetRandomProxyUseCase(reader, logger)
		result, err := uc.Execute(ctx, proxy.GetRandomProxyInput{})

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "1.1.1.1:8080", result.Address())
	})

	t.Run("returns error when no proxies", func(t *testing.T) {
		reader := mocks.NewReader(t)
		reader.EXPECT().
			GetAlive(ctx, float64(0), 0, mock.AnythingOfType("proxy.FilterOptions")).
			Return([]*proxy.Proxy{}, float64(0), 0, nil)

		uc := proxy.NewGetRandomProxyUseCase(reader, logger)
		_, err := uc.Execute(ctx, proxy.GetRandomProxyInput{})

		assert.ErrorIs(t, err, proxy.ErrNoProxiesAvailable)
	})

	t.Run("propagates reader error", func(t *testing.T) {
		reader := mocks.NewReader(t)
		reader.EXPECT().
			GetAlive(ctx, float64(0), 0, mock.AnythingOfType("proxy.FilterOptions")).
			Return(nil, float64(0), 0, errors.New("database error"))

		uc := proxy.NewGetRandomProxyUseCase(reader, logger)
		_, err := uc.Execute(ctx, proxy.GetRandomProxyInput{})

		assert.Error(t, err)
	})
}
