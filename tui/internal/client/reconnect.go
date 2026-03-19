// Package client provides reconnection logic for the clawmon daemon client.
package client

import (
	"fmt"
	"math"
	"time"
)

// Reconnection backoff constants.
const (
	InitialBackoff = 1 * time.Second
	MaxBackoff     = 30 * time.Second
	BackoffFactor  = 2.0
)

// ReconnectState tracks connection state and backoff for reconnection attempts.
type ReconnectState struct {
	// Attempts is the number of consecutive failed connection attempts.
	Attempts int
	// LastAttempt is the time of the last reconnection attempt.
	LastAttempt time.Time
	// Connected indicates whether the daemon is currently reachable.
	Connected bool
}

// NewReconnectState creates a new reconnect state, initially connected.
func NewReconnectState() *ReconnectState {
	return &ReconnectState{
		Connected: true,
	}
}

// MarkDisconnected records a failed connection attempt.
func (r *ReconnectState) MarkDisconnected() {
	r.Connected = false
	r.Attempts++
	r.LastAttempt = time.Now()
}

// MarkConnected resets state after a successful connection.
func (r *ReconnectState) MarkConnected() {
	r.Connected = true
	r.Attempts = 0
}

// NextBackoff returns the duration to wait before the next reconnection attempt.
func (r *ReconnectState) NextBackoff() time.Duration {
	if r.Attempts == 0 {
		return InitialBackoff
	}
	backoff := float64(InitialBackoff) * math.Pow(BackoffFactor, float64(r.Attempts-1))
	if backoff > float64(MaxBackoff) {
		backoff = float64(MaxBackoff)
	}
	return time.Duration(backoff)
}

// ShouldRetry returns true if enough time has elapsed since the last attempt.
func (r *ReconnectState) ShouldRetry() bool {
	if r.Connected {
		return false
	}
	if r.LastAttempt.IsZero() {
		return true
	}
	return time.Since(r.LastAttempt) >= r.NextBackoff()
}

// RetryMessage returns a human-readable status message for the reconnection state.
func (r *ReconnectState) RetryMessage() string {
	if r.Connected {
		return ""
	}
	remaining := r.NextBackoff() - time.Since(r.LastAttempt)
	if remaining < 0 {
		remaining = 0
	}
	secs := int(remaining.Seconds())
	if secs <= 0 {
		return "Reconnecting..."
	}
	return fmt.Sprintf("Daemon offline — retrying in %ds", secs)
}
