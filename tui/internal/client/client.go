// Package client provides an HTTP client for the clawmon daemon API.
package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/PeterHiroshi/clawmon/tui/internal/models"
)

const (
	// DefaultTimeout is the HTTP timeout for REST requests.
	DefaultTimeout = 5 * time.Second
)

// DaemonClient defines the interface for communicating with the clawmon daemon.
type DaemonClient interface {
	Health() (*models.HealthResponse, error)
	ListWorkspaces() ([]models.WorkspaceInfo, error)
	GetGitStatus(id string) (*models.GitStatus, error)
	GetTasks(id string) ([]models.TaskInfo, error)
	GetProcesses(id string) ([]models.ProcessInfo, error)
	GetEnvHealth(id string) (*models.EnvHealth, error)
	GetActivity(id string) ([]models.ActivityEvent, error)
	SubscribeEvents(ctx context.Context) (<-chan models.SseEvent, error)
}

// HTTPClient implements DaemonClient using HTTP.
type HTTPClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewHTTPClient creates a new HTTP client for the daemon API.
func NewHTTPClient(baseURL string) *HTTPClient {
	return &HTTPClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: DefaultTimeout,
		},
	}
}

// apiGet performs a GET request and decodes the JSON response.
func apiGet[T any](c *HTTPClient, path string) (*T, error) {
	url := c.baseURL + path
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("daemon unreachable at %s: %w", url, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response from %s: %w", path, err)
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("not found: %s", path)
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("daemon error (status %d) at %s: %s", resp.StatusCode, path, string(body))
	}

	var apiResp models.ApiResponse[T]
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("decoding response from %s: %w", path, err)
	}

	return &apiResp.Data, nil
}

// Health checks daemon health.
func (c *HTTPClient) Health() (*models.HealthResponse, error) {
	return apiGet[models.HealthResponse](c, "/health")
}

// ListWorkspaces returns all monitored workspaces.
func (c *HTTPClient) ListWorkspaces() ([]models.WorkspaceInfo, error) {
	result, err := apiGet[[]models.WorkspaceInfo](c, "/workspaces")
	if err != nil {
		return nil, err
	}
	return *result, nil
}

// GetGitStatus returns git sync status for a workspace.
func (c *HTTPClient) GetGitStatus(id string) (*models.GitStatus, error) {
	return apiGet[models.GitStatus](c, "/workspaces/"+id+"/git")
}

// GetTasks returns tasks for a workspace.
func (c *HTTPClient) GetTasks(id string) ([]models.TaskInfo, error) {
	result, err := apiGet[[]models.TaskInfo](c, "/workspaces/"+id+"/tasks")
	if err != nil {
		return nil, err
	}
	return *result, nil
}

// GetProcesses returns Claude Code processes for a workspace.
func (c *HTTPClient) GetProcesses(id string) ([]models.ProcessInfo, error) {
	result, err := apiGet[[]models.ProcessInfo](c, "/workspaces/"+id+"/processes")
	if err != nil {
		return nil, err
	}
	return *result, nil
}

// GetEnvHealth returns environment health for a workspace.
func (c *HTTPClient) GetEnvHealth(id string) (*models.EnvHealth, error) {
	return apiGet[models.EnvHealth](c, "/workspaces/"+id+"/env")
}

// GetActivity returns recent activity events for a workspace.
func (c *HTTPClient) GetActivity(id string) ([]models.ActivityEvent, error) {
	result, err := apiGet[[]models.ActivityEvent](c, "/workspaces/"+id+"/activity")
	if err != nil {
		return nil, err
	}
	return *result, nil
}

// SubscribeEvents opens an SSE connection to the daemon.
// Implemented in sse.go.
func (c *HTTPClient) SubscribeEvents(ctx context.Context) (<-chan models.SseEvent, error) {
	return subscribeSSE(ctx, c.baseURL+"/events")
}
