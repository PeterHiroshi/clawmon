package client

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewReconnectState(t *testing.T) {
	rs := NewReconnectState()
	assert.True(t, rs.Connected)
	assert.Equal(t, 0, rs.Attempts)
}

func TestMarkDisconnected(t *testing.T) {
	rs := NewReconnectState()
	rs.MarkDisconnected()
	assert.False(t, rs.Connected)
	assert.Equal(t, 1, rs.Attempts)
	assert.False(t, rs.LastAttempt.IsZero())
}

func TestMarkConnected(t *testing.T) {
	rs := NewReconnectState()
	rs.MarkDisconnected()
	rs.MarkDisconnected()
	rs.MarkConnected()
	assert.True(t, rs.Connected)
	assert.Equal(t, 0, rs.Attempts)
}

func TestNextBackoff(t *testing.T) {
	tests := []struct {
		name     string
		attempts int
		expected time.Duration
	}{
		{"zero attempts", 0, 1 * time.Second},
		{"first attempt", 1, 1 * time.Second},
		{"second attempt", 2, 2 * time.Second},
		{"third attempt", 3, 4 * time.Second},
		{"fourth attempt", 4, 8 * time.Second},
		{"fifth attempt", 5, 16 * time.Second},
		{"many attempts (capped)", 10, 30 * time.Second},
		{"very many attempts (capped)", 20, 30 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rs := &ReconnectState{Attempts: tt.attempts}
			assert.Equal(t, tt.expected, rs.NextBackoff())
		})
	}
}

func TestShouldRetry(t *testing.T) {
	t.Run("connected means no retry", func(t *testing.T) {
		rs := NewReconnectState()
		assert.False(t, rs.ShouldRetry())
	})

	t.Run("first disconnect allows immediate retry", func(t *testing.T) {
		rs := &ReconnectState{Connected: false}
		assert.True(t, rs.ShouldRetry())
	})

	t.Run("recent attempt means wait", func(t *testing.T) {
		rs := &ReconnectState{
			Connected:   false,
			Attempts:    1,
			LastAttempt: time.Now(),
		}
		assert.False(t, rs.ShouldRetry())
	})

	t.Run("old attempt allows retry", func(t *testing.T) {
		rs := &ReconnectState{
			Connected:   false,
			Attempts:    1,
			LastAttempt: time.Now().Add(-2 * time.Second),
		}
		assert.True(t, rs.ShouldRetry())
	})
}

func TestRetryMessage(t *testing.T) {
	t.Run("connected returns empty", func(t *testing.T) {
		rs := NewReconnectState()
		assert.Empty(t, rs.RetryMessage())
	})

	t.Run("disconnected with past backoff", func(t *testing.T) {
		rs := &ReconnectState{
			Connected:   false,
			Attempts:    1,
			LastAttempt: time.Now().Add(-5 * time.Second),
		}
		assert.Equal(t, "Reconnecting...", rs.RetryMessage())
	})

	t.Run("disconnected with pending backoff", func(t *testing.T) {
		rs := &ReconnectState{
			Connected:   false,
			Attempts:    5, // 16s backoff
			LastAttempt: time.Now(),
		}
		msg := rs.RetryMessage()
		assert.Contains(t, msg, "Daemon offline")
		assert.Contains(t, msg, "retrying in")
	})
}
