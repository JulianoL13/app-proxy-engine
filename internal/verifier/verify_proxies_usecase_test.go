package verifier_test

import (
	"context"
	"testing"
	"time"

	"github.com/JulianoL13/app-proxy-engine/internal/verifier"
	"github.com/JulianoL13/app-proxy-engine/internal/verifier/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestVerifyProxiesUseCase_Execute(t *testing.T) {
	// Setup
	mockChecker := mocks.NewProxyChecker(t)
	useCase := verifier.NewVerifyProxiesUseCase(mockChecker, 2)

	// Create mocks for proxies
	proxy1 := mocks.NewVerifiable(t)
	proxy1.On("Address").Return("1.1.1.1:8080")
	// Note: URL() is not called by UseCase, but by Checker.
	// Since we mock Checker, we don't strictly need to mock URL() unless argument matching calls it.

	proxy2 := mocks.NewVerifiable(t)
	proxy2.On("Address").Return("2.2.2.2:8080")

	proxy3 := mocks.NewVerifiable(t)
	proxy3.On("Address").Return("3.3.3.3:8080")

	proxies := []verifier.Verifiable{proxy1, proxy2, proxy3}

	expectedResultSuccess := verifier.Result{
		Success:   true,
		Latency:   100 * time.Millisecond,
		Anonymity: verifier.Elite,
	}

	expectedResultFail := verifier.Result{
		Success: false,
		Error:   assert.AnError,
	}

	// Expectations
	mockChecker.On("Verify", mock.Anything, proxy1).Return(expectedResultSuccess)
	mockChecker.On("Verify", mock.Anything, proxy2).Return(expectedResultFail)
	mockChecker.On("Verify", mock.Anything, proxy3).Return(expectedResultSuccess)

	// Execute
	results := useCase.Execute(context.Background(), proxies)

	// Assertions
	assert.Len(t, results, 3)
	assert.Equal(t, expectedResultSuccess, results["1.1.1.1:8080"])
	assert.Equal(t, expectedResultFail, results["2.2.2.2:8080"])
	assert.Equal(t, expectedResultSuccess, results["3.3.3.3:8080"])
}
