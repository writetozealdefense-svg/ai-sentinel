// Package detect is the HTTP client for detection-svc (Interface 1).
// Detection is delegated entirely to detection-svc; this package never
// reimplements detection logic.
package detect

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Finding mirrors a detection-svc finding (contracts/openapi/dlp.yaml).
type Finding struct {
	Type     string  `json:"type"`
	Category string  `json:"category"`
	Engine   string  `json:"engine"`
	Text     string  `json:"text"`
	Start    int     `json:"start"`
	End      int     `json:"end"`
	Score    float64 `json:"score"`
	RuleName *string `json:"ruleName"`
}

// Stats mirrors detection-svc scan stats.
type Stats struct {
	TotalFindings int            `json:"totalFindings"`
	ByCategory    map[string]int `json:"byCategory"`
	HighestScore  float64        `json:"highestScore"`
	DurationMs    int            `json:"durationMs"`
}

// ScanResult is the detection-svc /scan response.
type ScanResult struct {
	Findings        []Finding `json:"findings"`
	RedactedPreview string    `json:"redactedPreview"`
	Stats           Stats     `json:"stats"`
}

// scanRequest is the detection-svc /scan request body.
type scanRequest struct {
	Content     string `json:"content"`
	ContentType string `json:"contentType"`
	Filename    string `json:"filename"`
}

// Client calls detection-svc.
type Client struct {
	baseURL string
	http    *http.Client
}

// New returns a detection client targeting baseURL (e.g. http://detection-svc:8000).
func New(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		http:    &http.Client{Timeout: 30 * time.Second},
	}
}

// Scan posts content to detection-svc POST /scan and returns the result.
func (c *Client) Scan(ctx context.Context, content, contentType, filename string) (*ScanResult, error) {
	body, err := json.Marshal(scanRequest{Content: content, ContentType: contentType, Filename: filename})
	if err != nil {
		return nil, fmt.Errorf("marshal scan request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/scan", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build scan request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call detection-svc /scan: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("detection-svc /scan returned %d", resp.StatusCode)
	}

	var out ScanResult
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode scan response: %w", err)
	}
	if out.Stats.ByCategory == nil {
		out.Stats.ByCategory = map[string]int{}
	}
	return &out, nil
}

// Categories returns the distinct categories present in the scan, in a stable
// order derived from byCategory; falls back to scanning findings.
func (r *ScanResult) Categories() []string {
	seen := map[string]bool{}
	var cats []string
	// Prefer byCategory keys but keep deterministic order via findings.
	for _, f := range r.Findings {
		if f.Category != "" && !seen[f.Category] {
			seen[f.Category] = true
			cats = append(cats, f.Category)
		}
	}
	for cat := range r.Stats.ByCategory {
		if !seen[cat] {
			seen[cat] = true
			cats = append(cats, cat)
		}
	}
	return cats
}
