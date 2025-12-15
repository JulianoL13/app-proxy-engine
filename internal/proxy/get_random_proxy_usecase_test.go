package proxy_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/JulianoL13/app-proxy-engine/internal/proxy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type getRandomMockReader struct {
	proxies    []*proxy.Proxy
	nextCursor float64
	total      int
	err        error
}

func (m *getRandomMockReader) GetAlive(ctx context.Context, cursor float64, limit int) ([]*proxy.Proxy, float64, int, error) {
	return m.proxies, m.nextCursor, m.total, m.err
}

type getRandomTestLogger struct{}

func (l getRandomTestLogger) Info(msg string, args ...any)  {}
func (l getRandomTestLogger) Debug(msg string, args ...any) {}

func TestGetRandomProxyUseCase_Execute(t *testing.T) {
	ctx := context.Background()
	logger := getRandomTestLogger{}

	t.Run("returns random proxy", func(t *testing.T) {
		p1 := proxy.NewProxy("1.1.1.1", 8080, proxy.HTTP, "source1")
		p1.MarkSuccess(100*time.Millisecond, proxy.Elite)

		reader := &getRandomMockReader{
			proxies: []*proxy.Proxy{p1},
			total:   1,
		}

		uc := proxy.NewGetRandomProxyUseCase(reader, logger)
		result, err := uc.Execute(ctx)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "1.1.1.1:8080", result.Address())
	})

	t.Run("returns error when no proxies", func(t *testing.T) {
		reader := &getRandomMockReader{
			proxies: []*proxy.Proxy{},
		}

		uc := proxy.NewGetRandomProxyUseCase(reader, logger)
		_, err := uc.Execute(ctx)

		assert.ErrorIs(t, err, proxy.ErrNoProxiesAvailable)
	})

	t.Run("propagates reader error", func(t *testing.T) {
		reader := &getRandomMockReader{
			err: errors.New("database error"),
		}

		uc := proxy.NewGetRandomProxyUseCase(reader, logger)
		_, err := uc.Execute(ctx)

		assert.Error(t, err)
	})
}
