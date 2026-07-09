package copilot

// Live-mode stub (PHASE4-INTEGRATION.md "Live mode"): only reachable when
// COPILOT_LIVE=true AND COPILOT_API_KEY is set. The same evidence bundle is
// sent to the Claude Messages API with instructions to answer ONLY from it.
// The client is deliberately minimal — net/http, no SDK — and is never
// exercised in tests (no key in this environment); tests assert the gate.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	defaultLiveModel = "claude-sonnet-5"
	messagesURL      = "https://api.anthropic.com/v1/messages"
	anthropicVersion = "2023-06-01"
)

// LiveClient is the minimal Claude Messages API client. A nil *LiveClient
// means live mode is disabled — the offline composer is the only path.
type LiveClient struct {
	apiKey string
	model  string
	url    string
	httpc  *http.Client
}

// NewLiveClient returns a client only when COPILOT_LIVE=true AND
// COPILOT_API_KEY is set; otherwise nil (offline mode, the default). Model
// comes from COPILOT_MODEL, defaulting to claude-sonnet-5.
func NewLiveClient(getenv func(string) string) *LiveClient {
	if !strings.EqualFold(strings.TrimSpace(getenv("COPILOT_LIVE")), "true") {
		return nil
	}
	key := getenv("COPILOT_API_KEY")
	if key == "" {
		return nil
	}
	model := getenv("COPILOT_MODEL")
	if model == "" {
		model = defaultLiveModel
	}
	return &LiveClient{
		apiKey: key,
		model:  model,
		url:    messagesURL,
		httpc:  &http.Client{Timeout: 60 * time.Second},
	}
}

// Mode reports the healthz/AskResponse mode for a (possibly nil) live client.
func Mode(l *LiveClient) string {
	if l != nil {
		return "live"
	}
	return "offline"
}

// Model exposes the configured model (for startup logging).
func (l *LiveClient) Model() string { return l.model }

const liveSystemPrompt = `You are the ZealDefense AI Sentinel security copilot. ` +
	`Answer ONLY from the evidence bundle JSON provided by the user. ` +
	`Every fact in your answer must appear in the bundle; if the bundle does not contain the answer, say so. ` +
	`Never invent control text, identifiers, users, or numbers. ` +
	`Respond with a single JSON object with string keys "executive", "technical" and "compliance", ` +
	`each a short grounded narrative view of the incident.`

// Explain sends the question + evidence bundle to the Claude Messages API and
// parses the three views from the response. Citations remain the deterministic
// ones derived from the bundle by the caller.
func (l *LiveClient) Explain(ctx context.Context, question string, b *Bundle) (Views, error) {
	bundleJSON, err := json.Marshal(b)
	if err != nil {
		return Views{}, fmt.Errorf("marshal bundle: %w", err)
	}
	if question == "" {
		question = "Explain this violation."
	}
	body, err := json.Marshal(map[string]any{
		"model":      l.model,
		"max_tokens": 1024,
		"system":     liveSystemPrompt,
		"messages": []map[string]any{
			{"role": "user", "content": question + "\n\nEvidence bundle:\n" + string(bundleJSON)},
		},
	})
	if err != nil {
		return Views{}, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, l.url, bytes.NewReader(body))
	if err != nil {
		return Views{}, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("x-api-key", l.apiKey)
	req.Header.Set("anthropic-version", anthropicVersion)

	resp, err := l.httpc.Do(req)
	if err != nil {
		return Views{}, fmt.Errorf("messages api: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return Views{}, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return Views{}, fmt.Errorf("messages api status %d: %s", resp.StatusCode, truncate(string(raw), 300))
	}

	var parsed struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return Views{}, fmt.Errorf("decode response: %w", err)
	}
	var text string
	for _, blk := range parsed.Content {
		if blk.Type == "text" {
			text += blk.Text
		}
	}
	if text == "" {
		return Views{}, fmt.Errorf("messages api returned no text content")
	}

	var v Views
	if err := json.Unmarshal([]byte(extractJSONObject(text)), &v); err != nil || (v.Executive == "" && v.Technical == "" && v.Compliance == "") {
		// Model didn't return the requested JSON shape — surface the text as
		// the executive view rather than dropping the answer.
		return Views{Executive: strings.TrimSpace(text)}, nil
	}
	return v, nil
}

// extractJSONObject trims any prose around the first {...} block so a fenced
// or prefixed JSON answer still parses.
func extractJSONObject(s string) string {
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start >= 0 && end > start {
		return s[start : end+1]
	}
	return s
}
