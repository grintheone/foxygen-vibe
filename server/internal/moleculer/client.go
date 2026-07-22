package moleculer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const registryNodesPath = "/v1/registry/nodes"

type Config struct {
	Enabled bool
	URL     string
	Timeout time.Duration
}

func (c Config) Validate() error {
	if !c.Enabled {
		return nil
	}
	if strings.TrimSpace(c.URL) == "" {
		return errors.New("sidecar URL is required when Moleculer integration is enabled")
	}

	parsed, err := url.ParseRequestURI(c.URL)
	if err != nil {
		return fmt.Errorf("parse sidecar URL: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("sidecar URL scheme must be http or https, got %q", parsed.Scheme)
	}
	if parsed.Host == "" {
		return errors.New("sidecar URL must include a host")
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return errors.New("sidecar URL must not include a query or fragment")
	}
	if c.Timeout <= 0 {
		return errors.New("sidecar request timeout must be greater than zero")
	}

	return nil
}

type Client struct {
	baseURL    string
	httpClient *http.Client
}

type RegistryStatus struct {
	NodeCount int
}

func New(config Config) (*Client, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	if !config.Enabled {
		return nil, nil
	}

	return &Client{
		baseURL: strings.TrimRight(config.URL, "/"),
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}, nil
}

func (c *Client) ProbeRegistry(ctx context.Context) (RegistryStatus, error) {
	if c == nil {
		return RegistryStatus{}, errors.New("Moleculer sidecar client is not configured")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+registryNodesPath, nil)
	if err != nil {
		return RegistryStatus{}, fmt.Errorf("create registry request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return RegistryStatus{}, fmt.Errorf("request sidecar registry: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		message := strings.TrimSpace(string(body))
		if message == "" {
			message = resp.Status
		}
		return RegistryStatus{}, fmt.Errorf("sidecar registry returned %s: %s", resp.Status, message)
	}

	var nodes []json.RawMessage
	decoder := json.NewDecoder(io.LimitReader(resp.Body, 1024*1024))
	if err := decoder.Decode(&nodes); err != nil {
		return RegistryStatus{}, fmt.Errorf("decode sidecar registry response: %w", err)
	}

	return RegistryStatus{NodeCount: len(nodes)}, nil
}
