package proxy

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSmartCooldown(t *testing.T) {
	p := NewProxy("127.0.0.1", 8080, HTTP, "test")

	p.MarkFailure()
	assert.Equal(t, 1, p.FailCount, "Expected FailCount 1")

	expectedDuration := 5 * time.Minute
	remaining := time.Until(p.CooldownUntil)
	assert.InDelta(t, expectedDuration, remaining, float64(time.Second), "1st failure should be ~5m")
	assert.False(t, p.IsReady(), "Proxy should not be ready after failure")

	p.MarkFailure()
	expectedDuration = 10 * time.Minute
	remaining = time.Until(p.CooldownUntil)
	assert.InDelta(t, expectedDuration, remaining, float64(time.Second), "2nd failure should be ~10m")

	p.MarkFailure()
	expectedDuration = 20 * time.Minute
	remaining = time.Until(p.CooldownUntil)
	assert.InDelta(t, expectedDuration, remaining, float64(time.Second), "3rd failure should be ~20m")

	p.MarkSuccess(100*time.Millisecond, Elite)
	assert.Equal(t, 0, p.FailCount)
	assert.True(t, p.CooldownUntil.IsZero())
	assert.True(t, p.IsReady())
}
