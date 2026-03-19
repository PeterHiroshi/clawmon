package client

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSSESubscription(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming not supported", http.StatusInternalServerError)
			return
		}

		events := []struct {
			eventType string
			data      string
		}{
			{"git_status", `{"event_type":"git_status","workspace_id":"alpha","data":{"branch":"main"}}`},
			{"task_update", `{"event_type":"task_update","workspace_id":"alpha","data":{"name":"test"}}`},
			{"process_update", `{"event_type":"process_update","workspace_id":"beta","data":{"pid":123}}`},
		}

		for _, e := range events {
			fmt.Fprintf(w, "event:%s\ndata:%s\n\n", e.eventType, e.data)
			flusher.Flush()
		}
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ch, err := subscribeSSE(ctx, server.URL)
	require.NoError(t, err)

	// Collect events
	var received []string
	for i := 0; i < 3; i++ {
		select {
		case event, ok := <-ch:
			require.True(t, ok, "channel closed prematurely")
			received = append(received, event.EventType)
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for SSE event")
		}
	}

	assert.Equal(t, []string{"git_status", "task_update", "process_update"}, received)
}

func TestSSEContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, ok := w.(http.Flusher)
		if !ok {
			return
		}

		// Send one event then wait
		fmt.Fprintf(w, "event:test\ndata:{\"event_type\":\"test\",\"workspace_id\":null,\"data\":null}\n\n")
		flusher.Flush()

		// Block until client disconnects
		<-r.Context().Done()
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	ch, err := subscribeSSE(ctx, server.URL)
	require.NoError(t, err)

	// Read the first event
	select {
	case event := <-ch:
		assert.Equal(t, "test", event.EventType)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for first event")
	}

	// Cancel context — channel should close
	cancel()

	select {
	case _, ok := <-ch:
		if ok {
			// May get one more buffered event, but channel should eventually close
		}
	case <-time.After(2 * time.Second):
		t.Fatal("channel did not close after context cancel")
	}
}

func TestSSEServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	ctx := context.Background()
	_, err := subscribeSSE(ctx, server.URL)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "status 500")
}

func TestSSEWorkspaceID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)
		fmt.Fprintf(w, "data:{\"event_type\":\"update\",\"workspace_id\":\"my-ws\",\"data\":{}}\n\n")
		flusher.Flush()
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	ch, err := subscribeSSE(ctx, server.URL)
	require.NoError(t, err)

	select {
	case event := <-ch:
		require.NotNil(t, event.WorkspaceID)
		assert.Equal(t, "my-ws", *event.WorkspaceID)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out")
	}
}
