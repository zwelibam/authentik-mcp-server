package authentik

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"time"
)

// Response types for the endpoints we need in Phase 1+2

type User struct {
	PK        int     `json:"pk"`
	Username  string  `json:"username"`
	Email     string  `json:"email"`
	LastLogin *string `json:"last_login"`
	IsActive  bool    `json:"is_active"`
}

type Group struct {
	PK   string `json:"pk"`
	Name string `json:"name"`
}

type EventUser struct {
	Username string `json:"username"`
	Email    string `json:"email"`
}

type Event struct {
	PK       string     `json:"pk"`
	Action   string     `json:"action"`
	App      string     `json:"app"`
	DateTime string     `json:"created"`
	ClientIP string     `json:"client_ip"`
	User     *EventUser `json:"user"`
}

type Application struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type Config struct {
	ErrorReportingEnabled bool     `json:"error_reporting_enabled"`
	Capabilities          []string `json:"capabilities"`
}

type CreateUserRequest struct {
	Username string   `json:"username"`
	Name     string   `json:"name"`
	Email    string   `json:"email"`
	IsActive bool     `json:"is_active"`
	Groups   []string `json:"groups"`
	Path     string   `json:"path,omitempty"`
}

type SetPasswordRequest struct {
	Password string `json:"password"`
}

type paginatedResponse[T any] struct {
	Count   int `json:"count"`
	Results []T `json:"results"`
}

// Client is the Authentik API client.
type Client struct {
	http    *http.Client
	baseURL string
	token   string
}

// retryTransport retries on 5xx or network errors (3 attempts, 500ms backoff).
type retryTransport struct {
	base     http.RoundTripper
	maxTries int
}

func (r *retryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var resp *http.Response
	var err error
	for i := 0; i < r.maxTries; i++ {
		if i > 0 {
			time.Sleep(500 * time.Millisecond)
		}
		resp, err = r.base.RoundTrip(req)
		if err == nil && resp.StatusCode < 500 {
			return resp, nil
		}
	}
	return resp, err
}

// NewClientWrapper creates a new Authentik client from environment variables.
// Requires AUTHENTIK_URL and AUTHENTIK_TOKEN.
// Set AUTHENTIK_TLS_SKIP_VERIFY=true to disable TLS verification (default: verify).
func NewClientWrapper(ctx context.Context) (*Client, error) {
	baseURL := os.Getenv("AUTHENTIK_URL")
	token := os.Getenv("AUTHENTIK_TOKEN")
	if baseURL == "" {
		return nil, fmt.Errorf("AUTHENTIK_URL is required")
	}
	if token == "" {
		return nil, fmt.Errorf("AUTHENTIK_TOKEN is required")
	}

	skipVerify := os.Getenv("AUTHENTIK_TLS_SKIP_VERIFY") == "true"
	if skipVerify {
		slog.Warn("TLS verification disabled", "hint", "set AUTHENTIK_TLS_SKIP_VERIFY=false to enable")
	}

	transport := &retryTransport{
		base: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: skipVerify}, //nolint:gosec
		},
		maxTries: 3,
	}

	return &Client{
		http:    &http.Client{Transport: transport, Timeout: 30 * time.Second},
		baseURL: baseURL,
		token:   token,
	}, nil
}

// BaseURL returns the configured Authentik base URL.
func (c *Client) BaseURL() string { return c.baseURL }

func (c *Client) get(ctx context.Context, path string, params url.Values, out any) error {
	u := c.baseURL + path
	if len(params) > 0 {
		u += "?" + params.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("GET %s: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("GET %s: HTTP %d", path, resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *Client) post(ctx context.Context, path string, body any, out any) error {
	jsonBytes, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("POST %s: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("POST %s: HTTP %d", path, resp.StatusCode)
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}

// GetUsers searches for users by username.
func (c *Client) GetUsers(ctx context.Context, search string) ([]User, error) {
	var result paginatedResponse[User]
	err := c.get(ctx, "/api/v3/core/users/", url.Values{"search": {search}}, &result)
	return result.Results, err
}

// GetGroupsForUser returns groups the user belongs to.
func (c *Client) GetGroupsForUser(ctx context.Context, userPK int) ([]Group, error) {
	var result paginatedResponse[Group]
	err := c.get(ctx, "/api/v3/core/groups/", url.Values{"members_by_pk": {fmt.Sprint(userPK)}}, &result)
	return result.Results, err
}

// GetAllGroups returns all groups.
func (c *Client) GetAllGroups(ctx context.Context) ([]Group, error) {
	var result paginatedResponse[Group]
	err := c.get(ctx, "/api/v3/core/groups/", url.Values{"page_size": {"100"}}, &result)
	return result.Results, err
}

// GetGroupByName returns the group matching name exactly, or nil if absent.
func (c *Client) GetGroupByName(ctx context.Context, name string) (*Group, error) {
	var result paginatedResponse[Group]
	err := c.get(ctx, "/api/v3/core/groups/", url.Values{"search": {name}}, &result)
	if err != nil {
		return nil, err
	}
	for _, group := range result.Results {
		if group.Name == name {
			return &group, nil
		}
	}
	return nil, nil
}

// GetUserEvents returns the last N events for a user.
func (c *Client) GetUserEvents(ctx context.Context, userPK, pageSize int) ([]Event, error) {
	var result paginatedResponse[Event]
	err := c.get(ctx, "/api/v3/events/events/", url.Values{
		"user__pk":  {fmt.Sprint(userPK)},
		"page_size": {fmt.Sprint(pageSize)},
		"ordering":  {"-created"},
	}, &result)
	return result.Results, err
}

// GetEventsByAction returns events filtered by action type.
func (c *Client) GetEventsByAction(ctx context.Context, action string, pageSize int) ([]Event, error) {
	var result paginatedResponse[Event]
	err := c.get(ctx, "/api/v3/events/events/", url.Values{
		"action":    {action},
		"page_size": {fmt.Sprint(pageSize)},
		"ordering":  {"-created"},
	}, &result)
	return result.Results, err
}

// GetApplications returns all applications.
func (c *Client) GetApplications(ctx context.Context) ([]Application, error) {
	var result paginatedResponse[Application]
	err := c.get(ctx, "/api/v3/core/applications/", nil, &result)
	return result.Results, err
}

// GetConfig fetches the Authentik server configuration (for smoke test / version check).
func (c *Client) GetConfig(ctx context.Context) (*Config, error) {
	var cfg Config
	err := c.get(ctx, "/api/v3/root/config/", nil, &cfg)
	return &cfg, err
}

// CreateUser creates an Authentik user.
func (c *Client) CreateUser(ctx context.Context, req CreateUserRequest) (*User, error) {
	var user User
	err := c.post(ctx, "/api/v3/core/users/", req, &user)
	return &user, err
}

// SetUserPassword sets a user's password.
func (c *Client) SetUserPassword(ctx context.Context, userPK int, password string) error {
	return c.post(ctx, fmt.Sprintf("/api/v3/core/users/%d/set_password/", userPK), SetPasswordRequest{Password: password}, nil)
}

// AddUserToGroup adds a user to a group.
func (c *Client) AddUserToGroup(ctx context.Context, groupPK string, userPK int) error {
	return c.post(ctx, fmt.Sprintf("/api/v3/core/groups/%s/add_user/", groupPK), map[string]int{"pk": userPK}, nil)
}

// RemoveUserFromGroup removes a user from a group.
func (c *Client) RemoveUserFromGroup(ctx context.Context, groupPK string, userPK int) error {
	return c.post(ctx, fmt.Sprintf("/api/v3/core/groups/%s/remove_user/", groupPK), map[string]int{"pk": userPK}, nil)
}
