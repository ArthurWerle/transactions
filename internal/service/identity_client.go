package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// IdentityClient resolves user information owned by the identity service.
// Transactions store only a created_by_id; the display name lives in the
// identity service and is fetched on demand.
type IdentityClient interface {
	// GetUserName returns the display name for the given user ID. It returns
	// an error when the user cannot be resolved (not found, identity
	// unavailable, etc.); callers should treat the name as unknown rather
	// than fail their own request.
	GetUserName(ctx context.Context, id uint) (string, error)
}

// HTTPIdentityClient talks to the identity service over the internal docker
// network using its session-less public user lookup endpoint.
type HTTPIdentityClient struct {
	baseURL string
	client  *http.Client
}

// NewHTTPIdentityClient builds a client targeting the given identity base URL
// (e.g. http://identity:8080). A short timeout keeps a slow or unavailable
// identity service from blocking transaction reads.
func NewHTTPIdentityClient(baseURL string) *HTTPIdentityClient {
	return &HTTPIdentityClient{
		baseURL: baseURL,
		client:  &http.Client{Timeout: 3 * time.Second},
	}
}

func (c *HTTPIdentityClient) GetUserName(ctx context.Context, id uint) (string, error) {
	url := fmt.Sprintf("%s/api/v1/internal/users/%d", c.baseURL, id)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to build identity request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call identity service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("identity service returned status %d", resp.StatusCode)
	}

	var body struct {
		ID   uint   `json:"id"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", fmt.Errorf("failed to decode identity response: %w", err)
	}

	return body.Name, nil
}
