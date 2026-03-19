package client

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/PeterHiroshi/clawmon/tui/internal/models"
)

// subscribeSSE opens an SSE connection and returns a channel of events.
func subscribeSSE(ctx context.Context, url string) (<-chan models.SseEvent, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating SSE request: %w", err)
	}
	req.Header.Set("Accept", "text/event-stream")

	// Use a client with no timeout for SSE streaming.
	sseClient := &http.Client{}
	resp, err := sseClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("connecting to SSE stream at %s: %w", url, err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("SSE stream returned status %d", resp.StatusCode)
	}

	ch := make(chan models.SseEvent, 16)

	go func() {
		defer resp.Body.Close()
		defer close(ch)

		scanner := bufio.NewScanner(resp.Body)
		var eventType string
		var dataLines []string

		for scanner.Scan() {
			line := scanner.Text()

			if line == "" {
				// Empty line = flush event
				if len(dataLines) > 0 {
					data := strings.Join(dataLines, "\n")
					var event models.SseEvent
					if err := json.Unmarshal([]byte(data), &event); err == nil {
						if eventType != "" {
							event.EventType = eventType
						}
						select {
						case ch <- event:
						case <-ctx.Done():
							return
						}
					}
					eventType = ""
					dataLines = nil
				}
				continue
			}

			if strings.HasPrefix(line, "event:") {
				eventType = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
			} else if strings.HasPrefix(line, "data:") {
				dataLines = append(dataLines, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
			}
			// Ignore comments (lines starting with :) and unknown fields
		}
	}()

	return ch, nil
}
