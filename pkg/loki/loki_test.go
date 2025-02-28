package loki

import (
	"context"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

type MockLogger struct{}

func (m *MockLogger) Error(msg string, args ...any) {
}

func Test_ConfigValidation(t *testing.T) {
	cfg := Config{}
	_, err := New(context.Background(), cfg, &MockLogger{})
	assert.Error(t, err)

	cfg.Url = "SomeUrl"
	pusher, err := New(context.Background(), cfg, &MockLogger{})
	assert.NoError(t, err)
	assert.Equal(t, cfg.Url, pusher.config.Url)
	assert.Equal(t, 1000, pusher.config.BatchMaxSize)
	assert.Equal(t, 5*time.Second, pusher.config.BatchMaxWait)
	assert.Equal(t, map[string]string{}, pusher.config.Labels)
}
