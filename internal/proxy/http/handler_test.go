package http_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/JulianoL13/app-proxy-engine/internal/proxy"
	proxyhttp "github.com/JulianoL13/app-proxy-engine/internal/proxy/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testLogger struct{}

func (l testLogger) Debug(msg string, args ...any) {}
func (l testLogger) Info(msg string, args ...any)  {}
func (l testLogger) Warn(msg string, args ...any)  {}
func (l testLogger) Error(msg string, args ...any) {}
func (l testLogger) With(args ...any) proxyhttp.Logger {
	return l
}

type mockGetProxiesUseCase struct {
	proxies    []*proxy.Proxy
	nextCursor float64
	total      int
	err        error
}

func (m *mockGetProxiesUseCase) Execute(ctx context.Context, input proxyhttp.GetProxiesInput) (proxyhttp.GetProxiesOutput, error) {
	if m.err != nil {
		return proxyhttp.GetProxiesOutput{}, m.err
	}

	filtered := make([]*proxy.Proxy, 0)
	for _, p := range m.proxies {
		if input.Protocol != "" && string(p.Protocol) != input.Protocol {
			continue
		}
		if input.Anonymity != "" && string(p.Anonymity) != input.Anonymity {
			continue
		}
		filtered = append(filtered, p)
	}

	return proxyhttp.GetProxiesOutput{
		Proxies:    filtered,
		NextCursor: m.nextCursor,
		Total:      m.total, // In this mock we don't update Total, but it's enough for tests
	}, nil
}

type mockGetRandomProxyUseCase struct {
	proxy *proxy.Proxy
	err   error
}

func (m *mockGetRandomProxyUseCase) Execute(ctx context.Context) (*proxy.Proxy, error) {
	return m.proxy, m.err
}

func TestHandler_Health(t *testing.T) {
	logger := testLogger{}
	handler := proxyhttp.NewHandler(&mockGetProxiesUseCase{}, &mockGetRandomProxyUseCase{}, logger)
	router := proxyhttp.NewRouter(handler, logger)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "ok")
}

func TestHandler_GetProxies(t *testing.T) {
	p1 := proxy.NewProxy("1.1.1.1", 8080, proxy.HTTP, "source1")
	p1.MarkSuccess(100*time.Millisecond, proxy.Elite)

	p2 := proxy.NewProxy("2.2.2.2", 3128, proxy.SOCKS5, "source2")
	p2.MarkSuccess(200*time.Millisecond, proxy.Anonymous)

	logger := testLogger{}

	t.Run("returns all proxies", func(t *testing.T) {
		getProxiesUC := &mockGetProxiesUseCase{
			proxies: []*proxy.Proxy{p1, p2},
			total:   2,
		}

		handler := proxyhttp.NewHandler(getProxiesUC, &mockGetRandomProxyUseCase{}, logger)
		router := proxyhttp.NewRouter(handler, logger)

		req := httptest.NewRequest(http.MethodGet, "/proxies", nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var result proxyhttp.PaginatedResponse
		err := json.Unmarshal(rec.Body.Bytes(), &result)
		require.NoError(t, err)

		assert.Len(t, result.Data, 2)
		assert.Equal(t, 50, result.Limit)
		assert.Equal(t, 2, result.TotalCount)
	})

	t.Run("filters by protocol", func(t *testing.T) {
		getProxiesUC := &mockGetProxiesUseCase{
			proxies: []*proxy.Proxy{p1, p2},
			total:   2,
		}

		handler := proxyhttp.NewHandler(getProxiesUC, &mockGetRandomProxyUseCase{}, logger)
		router := proxyhttp.NewRouter(handler, logger)

		req := httptest.NewRequest(http.MethodGet, "/proxies?protocol=http", nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		var result proxyhttp.PaginatedResponse
		json.Unmarshal(rec.Body.Bytes(), &result)

		assert.Len(t, result.Data, 1)
		assert.Equal(t, "http", result.Data[0].Protocol)
	})

	t.Run("filters by anonymity", func(t *testing.T) {
		getProxiesUC := &mockGetProxiesUseCase{
			proxies: []*proxy.Proxy{p1, p2},
			total:   2,
		}

		handler := proxyhttp.NewHandler(getProxiesUC, &mockGetRandomProxyUseCase{}, logger)
		router := proxyhttp.NewRouter(handler, logger)

		req := httptest.NewRequest(http.MethodGet, "/proxies?anonymity=elite", nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		var result proxyhttp.PaginatedResponse
		json.Unmarshal(rec.Body.Bytes(), &result)

		assert.Len(t, result.Data, 1)
		assert.Equal(t, "elite", result.Data[0].Anonymity)
	})

	t.Run("filters by max latency", func(t *testing.T) {
		getProxiesUC := &mockGetProxiesUseCase{
			proxies: []*proxy.Proxy{p1, p2},
			total:   2,
		}

		handler := proxyhttp.NewHandler(getProxiesUC, &mockGetRandomProxyUseCase{}, logger)
		router := proxyhttp.NewRouter(handler, logger)

		req := httptest.NewRequest(http.MethodGet, "/proxies?max_latency_ms=150", nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		var result proxyhttp.PaginatedResponse
		json.Unmarshal(rec.Body.Bytes(), &result)

		assert.Len(t, result.Data, 1)
		assert.Equal(t, int64(100), result.Data[0].Latency)
	})
}

func TestHandler_GetRandomProxy(t *testing.T) {
	p1 := proxy.NewProxy("1.1.1.1", 8080, proxy.HTTP, "source1")
	p1.MarkSuccess(100*time.Millisecond, proxy.Elite)

	logger := testLogger{}

	t.Run("returns a proxy", func(t *testing.T) {
		getRandomUC := &mockGetRandomProxyUseCase{proxy: p1}

		handler := proxyhttp.NewHandler(&mockGetProxiesUseCase{}, getRandomUC, logger)
		router := proxyhttp.NewRouter(handler, logger)

		req := httptest.NewRequest(http.MethodGet, "/proxies/random", nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var result proxyhttp.ProxyResponse
		err := json.Unmarshal(rec.Body.Bytes(), &result)
		require.NoError(t, err)

		assert.Equal(t, "1.1.1.1:8080", result.Address)
	})

	t.Run("returns 404 when no proxies", func(t *testing.T) {
		getRandomUC := &mockGetRandomProxyUseCase{err: proxy.ErrNoProxiesAvailable}

		handler := proxyhttp.NewHandler(&mockGetProxiesUseCase{}, getRandomUC, logger)
		router := proxyhttp.NewRouter(handler, logger)

		req := httptest.NewRequest(http.MethodGet, "/proxies/random", nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}
