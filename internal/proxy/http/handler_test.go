package http_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	logmocks "github.com/JulianoL13/app-proxy-engine/internal/common/logs/mocks"
	"github.com/JulianoL13/app-proxy-engine/internal/proxy"
	proxyhttp "github.com/JulianoL13/app-proxy-engine/internal/proxy/http"
	"github.com/JulianoL13/app-proxy-engine/internal/proxy/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestHandler_Health(t *testing.T) {
	reader := mocks.NewReader(t)
	handler := proxyhttp.NewHandler(reader, logmocks.LoggerMock{})
	router := proxyhttp.NewRouter(handler, logmocks.LoggerMock{})

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

	t.Run("returns all proxies", func(t *testing.T) {
		reader := mocks.NewReader(t)
		reader.EXPECT().GetAlive(mock.Anything, float64(0), 50).
			Return([]*proxy.Proxy{p1, p2}, float64(0), 2, nil)

		handler := proxyhttp.NewHandler(reader, logmocks.LoggerMock{})
		router := proxyhttp.NewRouter(handler, logmocks.LoggerMock{})

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
		reader := mocks.NewReader(t)
		reader.EXPECT().GetAlive(mock.Anything, float64(0), 50).
			Return([]*proxy.Proxy{p1, p2}, float64(0), 2, nil)

		handler := proxyhttp.NewHandler(reader, logmocks.LoggerMock{})
		router := proxyhttp.NewRouter(handler, logmocks.LoggerMock{})

		req := httptest.NewRequest(http.MethodGet, "/proxies?protocol=http", nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		var result proxyhttp.PaginatedResponse
		json.Unmarshal(rec.Body.Bytes(), &result)

		assert.Len(t, result.Data, 1)
		assert.Equal(t, "http", result.Data[0].Protocol)
	})

	t.Run("filters by anonymity", func(t *testing.T) {
		reader := mocks.NewReader(t)
		reader.EXPECT().GetAlive(mock.Anything, float64(0), 50).
			Return([]*proxy.Proxy{p1, p2}, float64(0), 2, nil)

		handler := proxyhttp.NewHandler(reader, logmocks.LoggerMock{})
		router := proxyhttp.NewRouter(handler, logmocks.LoggerMock{})

		req := httptest.NewRequest(http.MethodGet, "/proxies?anonymity=elite", nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		var result proxyhttp.PaginatedResponse
		json.Unmarshal(rec.Body.Bytes(), &result)

		assert.Len(t, result.Data, 1)
		assert.Equal(t, "elite", result.Data[0].Anonymity)
	})

	t.Run("filters by max latency", func(t *testing.T) {
		reader := mocks.NewReader(t)
		reader.EXPECT().GetAlive(mock.Anything, float64(0), 50).
			Return([]*proxy.Proxy{p1, p2}, float64(0), 2, nil)

		handler := proxyhttp.NewHandler(reader, logmocks.LoggerMock{})
		router := proxyhttp.NewRouter(handler, logmocks.LoggerMock{})

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

	t.Run("returns a proxy", func(t *testing.T) {
		reader := mocks.NewReader(t)
		reader.EXPECT().GetAlive(mock.Anything, float64(0), 0).
			Return([]*proxy.Proxy{p1}, float64(0), 1, nil)

		handler := proxyhttp.NewHandler(reader, logmocks.LoggerMock{})
		router := proxyhttp.NewRouter(handler, logmocks.LoggerMock{})

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
		reader := mocks.NewReader(t)
		reader.EXPECT().GetAlive(mock.Anything, float64(0), 0).
			Return([]*proxy.Proxy{}, float64(0), 0, nil)

		handler := proxyhttp.NewHandler(reader, logmocks.LoggerMock{})
		router := proxyhttp.NewRouter(handler, logmocks.LoggerMock{})

		req := httptest.NewRequest(http.MethodGet, "/proxies/random", nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("applies filters to random", func(t *testing.T) {
		p2 := proxy.NewProxy("2.2.2.2", 3128, proxy.SOCKS5, "source2")
		p2.MarkSuccess(200*time.Millisecond, proxy.Anonymous)

		reader := mocks.NewReader(t)
		reader.EXPECT().GetAlive(mock.Anything, float64(0), 0).
			Return([]*proxy.Proxy{p1, p2}, float64(0), 2, nil)

		handler := proxyhttp.NewHandler(reader, logmocks.LoggerMock{})
		router := proxyhttp.NewRouter(handler, logmocks.LoggerMock{})

		req := httptest.NewRequest(http.MethodGet, "/proxies/random?protocol=socks5", nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		var result proxyhttp.ProxyResponse
		json.Unmarshal(rec.Body.Bytes(), &result)

		assert.Equal(t, "socks5", result.Protocol)
	})
}
