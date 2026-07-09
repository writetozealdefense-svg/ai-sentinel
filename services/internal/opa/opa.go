// Package opa is the HTTP client for the OPA decision endpoint (Interface 2):
// POST {OPA_URL}/v1/data/sentinel/dlp/decision -> {action, reasons}.
package opa

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Input is the OPA decision input (PHASE1-INTEGRATION.md Interface 2).
type Input struct {
	Provider      string         `json:"provider"`
	Sanctioned    bool           `json:"sanctioned"`
	Categories    []string       `json:"categories"`
	TotalFindings int            `json:"totalFindings"`
	HighestScore  float64        `json:"highestScore"`
	ByCategory    map[string]int `json:"byCategory"`
}

// Decision is the OPA result body.
type Decision struct {
	Action  string   `json:"action"`
	Reasons []string `json:"reasons"`
}

type opaRequest struct {
	Input Input `json:"input"`
}

type opaResponse struct {
	Result Decision `json:"result"`
}

// Client calls OPA.
type Client struct {
	baseURL string
	http    *http.Client
}

// New returns an OPA client targeting baseURL (e.g. http://opa:8181).
func New(baseURL string) *Client {
	return &Client{baseURL: baseURL, http: &http.Client{Timeout: 15 * time.Second}}
}

// Decide queries OPA and returns the decision. Action is one of
// ALLOW/WARN/REDACT/BLOCK.
func (c *Client) Decide(ctx context.Context, in Input) (*Decision, error) {
	if in.Categories == nil {
		in.Categories = []string{}
	}
	if in.ByCategory == nil {
		in.ByCategory = map[string]int{}
	}
	body, err := json.Marshal(opaRequest{Input: in})
	if err != nil {
		return nil, fmt.Errorf("marshal opa input: %w", err)
	}
	url := c.baseURL + "/v1/data/sentinel/dlp/decision"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build opa request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call opa: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("opa returned %d", resp.StatusCode)
	}

	var out opaResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode opa response: %w", err)
	}
	if out.Result.Action == "" {
		return nil, fmt.Errorf("opa returned empty action (policy not loaded?)")
	}
	return &out.Result, nil
}
