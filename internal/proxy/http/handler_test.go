package http_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	logmocks "github.com/JulianoL13/app-proxy-engine/internal/common/logs/mocks"
	"github.com/JulianoL13/app-proxy-engine/internal/proxy"
	proxyhttp "github.com/JulianoL13/app-proxy-engine/internal/proxy/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockReader implements http.Reader for testing
type MockReader struct {
	proxies []*proxy.Proxy
	err     error
}

func (m *MockReader) GetAlive(ctx context.Context) ([]*proxy.Proxy, error) {
	return m.proxies, m.err
}

func TestHandler_Health(t *testing.T) {
	handler := proxyhttp.NewHandler(&MockReader{}, logmocks.LoggerMock{})
	router := proxyhttp.NewRouter(handler)

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

	reader := &MockReader{proxies: []*proxy.Proxy{p1, p2}}
	handler := proxyhttp.NewHandler(reader, logmocks.LoggerMock{})
	router := proxyhttp.NewRouter(handler)

	t.Run("returns all proxies", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/proxies", nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var result []proxyhttp.ProxyResponse
		err := json.Unmarshal(rec.Body.Bytes(), &result)
		require.NoError(t, err)

		assert.Len(t, result, 2)
	})

	t.Run("filters by protocol", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/proxies?protocol=http", nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		var result []proxyhttp.ProxyResponse
		json.Unmarshal(rec.Body.Bytes(), &result)

		assert.Len(t, result, 1)
		assert.Equal(t, "http", result[0].Protocol)
	})

	t.Run("filters by anonymity", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/proxies?anonymity=elite", nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		var result []proxyhttp.ProxyResponse
		json.Unmarshal(rec.Body.Bytes(), &result)

		assert.Len(t, result, 1)
		assert.Equal(t, "elite", result[0].Anonymity)
	})

	t.Run("filters by max latency", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/proxies?max_latency_ms=150", nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		var result []proxyhttp.ProxyResponse
		json.Unmarshal(rec.Body.Bytes(), &result)

		assert.Len(t, result, 1)
		assert.Equal(t, int64(100), result[0].Latency)
	})
}

func TestHandler_GetRandomProxy(t *testing.T) {
	p1 := proxy.NewProxy("1.1.1.1", 8080, proxy.HTTP, "source1")
	p1.MarkSuccess(100*time.Millisecond, proxy.Elite)

	t.Run("returns a proxy", func(t *testing.T) {
		reader := &MockReader{proxies: []*proxy.Proxy{p1}}
		handler := proxyhttp.NewHandler(reader, logmocks.LoggerMock{})
		router := proxyhttp.NewRouter(handler)

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
		reader := &MockReader{proxies: []*proxy.Proxy{}}
		handler := proxyhttp.NewHandler(reader, logmocks.LoggerMock{})
		router := proxyhttp.NewRouter(handler)

		req := httptest.NewRequest(http.MethodGet, "/proxies/random", nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("applies filters to random", func(t *testing.T) {
		p2 := proxy.NewProxy("2.2.2.2", 3128, proxy.SOCKS5, "source2")
		p2.MarkSuccess(200*time.Millisecond, proxy.Anonymous)

		reader := &MockReader{proxies: []*proxy.Proxy{p1, p2}}
		handler := proxyhttp.NewHandler(reader, logmocks.LoggerMock{})
		router := proxyhttp.NewRouter(handler)

		req := httptest.NewRequest(http.MethodGet, "/proxies/random?protocol=socks5", nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		var result proxyhttp.ProxyResponse
		json.Unmarshal(rec.Body.Bytes(), &result)

		assert.Equal(t, "socks5", result.Protocol)
	})
}
