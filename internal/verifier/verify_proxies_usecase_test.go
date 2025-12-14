package verifier_test

import (
	"context"
	"testing"
	"time"

	logmocks "github.com/JulianoL13/app-proxy-engine/internal/common/logs/mocks"
	"github.com/JulianoL13/app-proxy-engine/internal/common/workerpool"
	"github.com/JulianoL13/app-proxy-engine/internal/verifier"
	"github.com/JulianoL13/app-proxy-engine/internal/verifier/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestVerifyProxiesUseCase_Execute(t *testing.T) {
	ctx := context.Background()

	t.Run("success with multiple proxies", func(t *testing.T) {
		mockChecker := mocks.NewProxyChecker(t)

		proxy1 := mocks.NewVerifiable(t)
		proxy1.On("Address").Return("1.1.1.1:8080")

		proxy2 := mocks.NewVerifiable(t)
		proxy2.On("Address").Return("2.2.2.2:8080")

		proxies := []verifier.Verifiable{proxy1, proxy2}

		expectedResult := verifier.VerifyOutput{
			Success:   true,
			Latency:   100 * time.Millisecond,
			Anonymity: "elite",
		}

		mockChecker.On("Verify", mock.Anything, proxy1).Return(expectedResult)
		mockChecker.On("Verify", mock.Anything, proxy2).Return(expectedResult)

		pool := workerpool.New(2)
		pool.Start()
		defer pool.Stop()

		uc := verifier.NewVerifyProxiesUseCase(mockChecker, pool, logmocks.LoggerMock{})
		resultChan := uc.Execute(ctx, proxies)

		results := make(map[string]verifier.VerifyOutput)
		for r := range resultChan {
			results[r.Address] = r.Output
		}

		assert.Len(t, results, 2)
		assert.Equal(t, expectedResult, results["1.1.1.1:8080"])
		assert.Equal(t, expectedResult, results["2.2.2.2:8080"])
		mockChecker.AssertExpectations(t)
	})

	t.Run("handles verification failure", func(t *testing.T) {
		mockChecker := mocks.NewProxyChecker(t)

		proxy := mocks.NewVerifiable(t)
		proxy.On("Address").Return("3.3.3.3:8080")

		failedResult := verifier.VerifyOutput{
			Success: false,
			Error:   assert.AnError,
		}

		mockChecker.On("Verify", mock.Anything, proxy).Return(failedResult)

		pool := workerpool.New(1)
		pool.Start()
		defer pool.Stop()

		uc := verifier.NewVerifyProxiesUseCase(mockChecker, pool, logmocks.LoggerMock{})
		resultChan := uc.Execute(ctx, []verifier.Verifiable{proxy})

		results := make(map[string]verifier.VerifyOutput)
		for r := range resultChan {
			results[r.Address] = r.Output
		}

		assert.Len(t, results, 1)
		assert.False(t, results["3.3.3.3:8080"].Success)
		mockChecker.AssertExpectations(t)
	})
}
