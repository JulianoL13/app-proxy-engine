package httpverifier

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type mockLogger struct{}

func (m *mockLogger) Info(msg string, args ...any)  {}
func (m *mockLogger) Warn(msg string, args ...any)  {}
func (m *mockLogger) Debug(msg string, args ...any) {}

func TestChecker_checkIntegrity(t *testing.T) {
	logger := &mockLogger{}

	tests := []struct {
		name     string
		baseline []byte
		body     []byte
		want     bool
	}{
		{
			name:     "valid httpbin response",
			baseline: []byte(`{"args":{},"headers":{"Host":"httpbin.org"},"origin":"1.2.3.4","url":"https://httpbin.org/get"}`),
			body:     []byte(`{"args":{},"headers":{"Host":"httpbin.org"},"origin":"5.6.7.8","url":"https://httpbin.org/get"}`),
			want:     true,
		},
		{
			name:     "payload too large",
			baseline: nil,
			body:     make([]byte, maxPayloadSize+1),
			want:     false,
		},
		{
			name:     "unexpected field injected",
			baseline: nil,
			body:     []byte(`{"args":{},"headers":{},"origin":"1.2.3.4","url":"https://httpbin.org/get","injected":"malicious"}`),
			want:     false,
		},
		{
			name:     "invalid json",
			baseline: nil,
			body:     []byte(`not valid json`),
			want:     false,
		},
		{
			name:     "suspicious X- header injected",
			baseline: []byte(`{"args":{},"headers":{"Host":"httpbin.org"},"origin":"1.2.3.4","url":"https://httpbin.org/get"}`),
			body:     []byte(`{"args":{},"headers":{"Host":"httpbin.org","X-Injected":"bad"},"origin":"1.2.3.4","url":"https://httpbin.org/get"}`),
			want:     false,
		},
		{
			name:     "new header without suspicious pattern passes",
			baseline: []byte(`{"args":{},"headers":{"Host":"httpbin.org"},"origin":"1.2.3.4","url":"https://httpbin.org/get"}`),
			body:     []byte(`{"args":{},"headers":{"Host":"httpbin.org","Accept":"*/*"},"origin":"1.2.3.4","url":"https://httpbin.org/get"}`),
			want:     true,
		},
		{
			name:     "empty body",
			baseline: nil,
			body:     []byte(`{}`),
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Checker{
				logger:   logger,
				baseline: tt.baseline,
			}

			got := c.checkIntegrity(tt.body)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestChecker_hashPayload(t *testing.T) {
	c := &Checker{}

	hash1 := c.hashPayload([]byte("test data"))
	hash2 := c.hashPayload([]byte("test data"))
	assert.Equal(t, hash1, hash2)
	hash3 := c.hashPayload([]byte("different data"))
	assert.NotEqual(t, hash1, hash3)

	assert.Len(t, hash1, 64)
}

func TestExpectedFields(t *testing.T) {
	assert.True(t, expectedFields["args"])
	assert.True(t, expectedFields["headers"])
	assert.True(t, expectedFields["origin"])
	assert.True(t, expectedFields["url"])
	assert.False(t, expectedFields["injected"])
	assert.False(t, expectedFields["script"])
}

func TestMaxPayloadSize(t *testing.T) {
	assert.Equal(t, 2048, maxPayloadSize)
}
