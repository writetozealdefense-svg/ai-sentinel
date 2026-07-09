package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zealdefense/ai-sentinel/services/internal/runtime"
)

// sidekickSample mirrors a real falcosidekick webhook body ("Terminal shell in
// container", priority Notice, k8s output_fields).
const sidekickSample = `{
  "uuid": "9f2b1c1e-6b7a-4b3e-9f6d-2f1a8c4d5e6f",
  "output": "10:23:45.123456789: Notice A shell was spawned in a container with an attached terminal (user=root k8s.ns=ai-sentinel k8s.pod=prompt-svc-7d9f container=prompt-svc shell=bash)",
  "priority": "Notice",
  "rule": "Terminal shell in container",
  "time": "2026-07-03T10:23:45.123456789Z",
  "output_fields": {
    "container.name": "prompt-svc",
    "k8s.ns.name": "ai-sentinel",
    "k8s.pod.name": "prompt-svc-7d9f",
    "proc.cmdline": "bash"
  },
  "source": "syscall",
  "tags": ["container", "shell", "mitre_execution"]
}`

func newTestServer(mode string) *server {
	return &server{buf: runtime.NewBuffer(), mode: mode}
}

func do(t *testing.T, s *server, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	s.router().ServeHTTP(w, req)
	return w
}

// TestHealthzExposesMode verifies /healthz includes the runtime mode.
func TestHealthzExposesMode(t *testing.T) {
	for _, mode := range []string{modeSimulate, modeIngest} {
		w := do(t, newTestServer(mode), http.MethodGet, "/healthz", "")
		if w.Code != http.StatusOK {
			t.Fatalf("mode %s: /healthz status = %d", mode, w.Code)
		}
		var got map[string]string
		if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
			t.Fatalf("mode %s: /healthz body: %v", mode, err)
		}
		if got["status"] != "ok" || got["service"] != serviceName || got["mode"] != mode {
			t.Fatalf("mode %s: /healthz = %v", mode, got)
		}
	}
}

// TestPostRejectedInSimulateMode verifies the compose path cannot be polluted:
// POST /runtime/events returns 409 {"error":"ingest disabled",...}.
func TestPostRejectedInSimulateMode(t *testing.T) {
	s := newTestServer(modeSimulate)
	w := do(t, s, http.MethodPost, "/runtime/events", sidekickSample)
	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", w.Code)
	}
	var got map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("body: %v", err)
	}
	if got["error"] != "ingest disabled" || got["detail"] != "RUNTIME_MODE=simulate" {
		t.Fatalf("body = %v", got)
	}
	if s.buf.Len() != 0 {
		t.Fatalf("rejected POST stored %d events", s.buf.Len())
	}
}

// TestIngestFalcoAlert verifies POST → 202 and GET returns the normalized
// event with simulated:false in ingest mode.
func TestIngestFalcoAlert(t *testing.T) {
	s := newTestServer(modeIngest)

	w := do(t, s, http.MethodPost, "/runtime/events", sidekickSample)
	if w.Code != http.StatusAccepted {
		t.Fatalf("POST status = %d, want 202 (body %s)", w.Code, w.Body.String())
	}

	w = do(t, s, http.MethodGet, "/runtime/events", "")
	if w.Code != http.StatusOK {
		t.Fatalf("GET status = %d", w.Code)
	}
	var events []runtime.Event
	if err := json.Unmarshal(w.Body.Bytes(), &events); err != nil {
		t.Fatalf("GET body: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}
	e := events[0]
	if e.Simulated {
		t.Fatal("ingested event returned with simulated:true")
	}
	if e.Rule != "Terminal shell in container" || e.Priority != "notice" || e.Source != "falco" {
		t.Fatalf("normalized event = %+v", e)
	}
	if e.Pod != "prompt-svc-7d9f" || e.Container != "prompt-svc" || e.Namespace != "ai-sentinel" {
		t.Fatalf("k8s fields = %+v", e)
	}
}

// TestIngestArray verifies a JSON array of alerts is accepted (robustness bonus).
func TestIngestArray(t *testing.T) {
	s := newTestServer(modeIngest)
	w := do(t, s, http.MethodPost, "/runtime/events", "["+sidekickSample+","+sidekickSample+"]")
	if w.Code != http.StatusAccepted {
		t.Fatalf("POST status = %d, want 202 (body %s)", w.Code, w.Body.String())
	}
	var got map[string]int
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("body: %v", err)
	}
	if got["accepted"] != 2 {
		t.Fatalf("accepted = %d, want 2", got["accepted"])
	}
	if s.buf.Len() != 2 {
		t.Fatalf("buffered %d events, want 2", s.buf.Len())
	}
}

// TestIngestMalformedJSON verifies malformed bodies return 400 {"error",...}.
func TestIngestMalformedJSON(t *testing.T) {
	s := newTestServer(modeIngest)
	for _, body := range []string{"{not json", "", "42", `{"uuid":"x"}`} {
		w := do(t, s, http.MethodPost, "/runtime/events", body)
		if w.Code != http.StatusBadRequest {
			t.Errorf("POST %q status = %d, want 400", body, w.Code)
			continue
		}
		var got map[string]string
		if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
			t.Errorf("POST %q body: %v", body, err)
		}
		if got["error"] == "" {
			t.Errorf("POST %q body missing error field: %v", body, got)
		}
	}
	if s.buf.Len() != 0 {
		t.Fatalf("malformed POSTs stored %d events", s.buf.Len())
	}
}

// TestIngestBodyTooLarge verifies bodies over 1MiB return 413.
func TestIngestBodyTooLarge(t *testing.T) {
	s := newTestServer(modeIngest)
	big := `{"rule":"x","output":"` + strings.Repeat("A", maxIngestBody) + `"}`
	w := do(t, s, http.MethodPost, "/runtime/events", big)
	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413", w.Code)
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("payload too large")) {
		t.Fatalf("body = %s", w.Body.String())
	}
}

// TestSimulateModeGETUnchanged verifies GET /runtime/events still serves the
// synthetic feed (simulated:true) exactly as before in simulate mode.
func TestSimulateModeGETUnchanged(t *testing.T) {
	s := newTestServer(modeSimulate)
	for i := 0; i < 5; i++ {
		s.buf.Generate()
	}
	w := do(t, s, http.MethodGet, "/runtime/events?limit=3", "")
	if w.Code != http.StatusOK {
		t.Fatalf("GET status = %d", w.Code)
	}
	var events []runtime.Event
	if err := json.Unmarshal(w.Body.Bytes(), &events); err != nil {
		t.Fatalf("GET body: %v", err)
	}
	if len(events) != 3 {
		t.Fatalf("got %d events, want 3", len(events))
	}
	for _, e := range events {
		if !e.Simulated {
			t.Fatal("simulate-mode event has simulated:false")
		}
	}
}
