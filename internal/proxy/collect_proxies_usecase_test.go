package proxy_test

import (
	"context"
	"testing"

	logmocks "github.com/JulianoL13/app-proxy-engine/internal/common/logs/mocks"
	"github.com/JulianoL13/app-proxy-engine/internal/proxy"
	"github.com/JulianoL13/app-proxy-engine/internal/proxy/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCollectProxiesUseCase_Execute(t *testing.T) {
	ctx := context.Background()

	t.Run("collects alive proxies", func(t *testing.T) {
		mockSource := mocks.NewProxySource(t)
		mockChecker := mocks.NewProxyChecker(t)

		p1 := mocks.NewProxyDataInput(t)
		p1.On("IP").Return("1.1.1.1")
		p1.On("Port").Return(8080)
		p1.On("Protocol").Return("http")
		p1.On("Source").Return("test")

		p2 := mocks.NewProxyDataInput(t)
		p2.On("IP").Return("2.2.2.2")
		p2.On("Port").Return(3128)
		p2.On("Protocol").Return("http")
		p2.On("Source").Return("test")

		mockSource.On("Fetch", mock.Anything).Return(
			[]proxy.ProxyDataInput{p1, p2},
			[]error{},
		)

		resultChan := make(chan proxy.CheckStreamResult, 2)
		resultChan <- proxy.CheckStreamResult{
			Address: "1.1.1.1:8080",
			Output:  proxy.CheckOutput{Success: true, Latency: 100},
		}
		resultChan <- proxy.CheckStreamResult{
			Address: "2.2.2.2:3128",
			Output:  proxy.CheckOutput{Success: false},
		}
		close(resultChan)

		mockChecker.On("Check", mock.Anything, mock.Anything).Return((<-chan proxy.CheckStreamResult)(resultChan))

		uc := proxy.NewCollectProxiesUseCase(mockSource, mockChecker, nil, logmocks.LoggerMock{})
		stream, err := uc.Execute(ctx)

		assert.NoError(t, err)

		var alive []*proxy.Proxy
		for p := range stream {
			alive = append(alive, p)
		}

		assert.Len(t, alive, 1)
		assert.Equal(t, "1.1.1.1:8080", alive[0].Address())
	})

	t.Run("returns empty when no proxies", func(t *testing.T) {
		mockSource := mocks.NewProxySource(t)
		mockChecker := mocks.NewProxyChecker(t)

		mockSource.On("Fetch", mock.Anything).Return([]proxy.ProxyDataInput{}, []error{})

		uc := proxy.NewCollectProxiesUseCase(mockSource, mockChecker, nil, logmocks.LoggerMock{})
		stream, err := uc.Execute(ctx)

		assert.NoError(t, err)

		var alive []*proxy.Proxy
		for p := range stream {
			alive = append(alive, p)
		}

		assert.Empty(t, alive)
	})
}
