package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/mcp-scooter/scooter/internal/domain/profile"
	"github.com/mcp-scooter/scooter/internal/domain/registry"
)

type ControlClient struct {
	baseURL string
	apiKey  string
	client  *http.Client
	timeout time.Duration
}

func NewControlClient(baseURL, apiKey string, timeout time.Duration) *ControlClient {
	return &ControlClient{
		baseURL: baseURL,
		apiKey:  apiKey,
		client: &http.Client{
			Timeout: timeout,
		},
		timeout: timeout,
	}
}

func (c *ControlClient) ListProfiles() ([]profile.Profile, error) {
	var profiles []profile.Profile
	err := c.get("/api/profiles", &profiles)
	return profiles, err
}

func (c *ControlClient) GetProfile(id string) (*profile.Profile, error) {
	var p profile.Profile
	err := c.get(fmt.Sprintf("/api/profiles/%s", id), &p)
	return &p, err
}

func (c *ControlClient) ListTools() ([]registry.Tool, error) {
	var tools []registry.Tool
	err := c.get("/api/tools", &tools)
	return tools, err
}

func (c *ControlClient) FindTools(query string) ([]registry.MCPEntry, error) {
	var entries []registry.MCPEntry
	err := c.get(fmt.Sprintf("/api/registry?q=%s", query), &entries)
	return entries, err
}

func (c *ControlClient) ActivateTool(server string, profileID string) error {
	body := map[string]string{
		"server":  server,
		"profile": profileID,
	}
	return c.post("/api/tools/activate", body, nil)
}

type CallResult struct {
	Content []ContentBlock `json:"content"`
	IsError bool           `json:"isError"`
}

type ContentBlock struct {
	Type string      `json:"type"`
	Text string      `json:"text,omitempty"`
	Data interface{} `json:"data,omitempty"`
}

func (c *ControlClient) CallTool(server, tool string, args map[string]interface{}, profileID string) (*CallResult, error) {
	body := map[string]interface{}{
		"server":    server,
		"tool":      tool,
		"arguments": args,
		"profile":   profileID,
	}
	var result CallResult
	err := c.post("/api/tools/call", body, &result)
	return &result, err
}

type Status struct {
	Running       bool     `json:"running"`
	Version       string   `json:"version"`
	Uptime        string   `json:"uptime"`
	ActiveProfile string   `json:"activeProfile"`
	ActiveServers []string `json:"activeServers"`
	Ports         struct {
		Control int `json:"control"`
		Gateway int `json:"gateway"`
	} `json:"ports"`
}

func (c *ControlClient) GetStatus() (*Status, error) {
	var status Status
	err := c.get("/api/status", &status)
	return &status, err
}

func (c *ControlClient) get(path string, v interface{}) error {
	req, err := http.NewRequest("GET", c.baseURL+path, nil)
	if err != nil {
		return err
	}
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return json.NewDecoder(resp.Body).Decode(v)
}

func (c *ControlClient) post(path string, body interface{}, v interface{}) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", c.baseURL+path, bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	if v != nil {
		return json.NewDecoder(resp.Body).Decode(v)
	}
	return nil
}
