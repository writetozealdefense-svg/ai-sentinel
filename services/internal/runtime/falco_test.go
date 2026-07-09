package runtime

import (
	"strings"
	"testing"
	"time"
)

// sidekickSample is a realistic falcosidekick HTTP-webhook payload (one JSON
// object per alert) for the classic "Terminal shell in container" rule.
const sidekickSample = `{
  "uuid": "9f2b1c1e-6b7a-4b3e-9f6d-2f1a8c4d5e6f",
  "output": "10:23:45.123456789: Notice A shell was spawned in a container with an attached terminal (user=root k8s.ns=ai-sentinel k8s.pod=prompt-svc-7d9f container=prompt-svc shell=bash parent=runc cmdline=bash)",
  "priority": "Notice",
  "rule": "Terminal shell in container",
  "time": "2026-07-03T10:23:45.123456789Z",
  "output_fields": {
    "container.name": "prompt-svc",
    "evt.time": 1782123825123456789,
    "k8s.ns.name": "ai-sentinel",
    "k8s.pod.name": "prompt-svc-7d9f",
    "proc.cmdline": "bash",
    "user.name": "root"
  },
  "source": "syscall",
  "tags": ["container", "shell", "mitre_execution"],
  "hostname": "ai-sentinel-control-plane"
}`

// TestNormalizeFalcoSample verifies a real falcosidekick payload normalizes to
// the RuntimeEvent shape with Simulated=false.
func TestNormalizeFalcoSample(t *testing.T) {
	alerts, err := ParseFalcoPayload([]byte(sidekickSample))
	if err != nil {
		t.Fatalf("ParseFalcoPayload: %v", err)
	}
	if len(alerts) != 1 {
		t.Fatalf("got %d alerts, want 1", len(alerts))
	}
	e := alerts[0].Normalize()

	if e.Simulated {
		t.Fatalf("ingested Falco event has Simulated=true; real events must be Simulated=false")
	}
	if e.EventID != "9f2b1c1e-6b7a-4b3e-9f6d-2f1a8c4d5e6f" {
		t.Fatalf("eventId = %q, want the sidekick uuid", e.EventID)
	}
	if e.Source != "falco" {
		t.Fatalf("source = %q, want falco (sidekick source %q is not part of the enum)", e.Source, "syscall")
	}
	if e.Rule != "Terminal shell in container" {
		t.Fatalf("rule = %q", e.Rule)
	}
	if e.Priority != "notice" {
		t.Fatalf("priority = %q, want notice", e.Priority)
	}
	if !strings.Contains(e.Message, "A shell was spawned in a container") {
		t.Fatalf("message = %q, want the Falco output line", e.Message)
	}
	if e.Namespace != "ai-sentinel" || e.Pod != "prompt-svc-7d9f" || e.Container != "prompt-svc" {
		t.Fatalf("k8s fields = ns %q pod %q container %q", e.Namespace, e.Pod, e.Container)
	}
	want := time.Date(2026, 7, 3, 10, 23, 45, 123456789, time.UTC)
	if !e.Timestamp.Equal(want) {
		t.Fatalf("timestamp = %v, want %v", e.Timestamp, want)
	}
	if e.Timestamp.Location().String() != "UTC" {
		t.Fatalf("timestamp not UTC: %v", e.Timestamp.Location())
	}
}

// TestMapFalcoPriority checks the full Falco priority → RuntimeEvent enum table.
func TestMapFalcoPriority(t *testing.T) {
	cases := map[string]string{
		"Emergency":     "critical",
		"Alert":         "critical",
		"Critical":      "critical",
		"Error":         "error",
		"Warning":       "warning",
		"Notice":        "notice",
		"Informational": "info",
		"Info":          "info",
		"Debug":         "info",
		"WARNING":       "warning", // case-insensitive
		"notice":        "notice",
		"":              "info", // unknown/absent falls back to info
		"bogus":         "info",
	}
	for in, want := range cases {
		if got := MapFalcoPriority(in); got != want {
			t.Errorf("MapFalcoPriority(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestNormalizeEventIDFallback verifies a missing uuid yields a deterministic
// non-empty eventId derived from output+time.
func TestNormalizeEventIDFallback(t *testing.T) {
	a := FalcoAlert{
		Output:   "Notice A shell was spawned",
		Priority: "Notice",
		Rule:     "Terminal shell in container",
		Time:     "2026-07-03T10:23:45Z",
	}
	e1 := a.Normalize()
	e2 := a.Normalize()
	if e1.EventID == "" {
		t.Fatal("fallback eventId is empty")
	}
	if e1.EventID != e2.EventID {
		t.Fatalf("fallback eventId not deterministic: %q vs %q", e1.EventID, e2.EventID)
	}
	a2 := a
	a2.Output = "different output"
	if a2.Normalize().EventID == e1.EventID {
		t.Fatal("different alerts produced the same fallback eventId")
	}
}

// TestParseFalcoPayloadArray verifies the array robustness bonus.
func TestParseFalcoPayloadArray(t *testing.T) {
	body := "[" + sidekickSample + "," + sidekickSample + "]"
	alerts, err := ParseFalcoPayload([]byte(body))
	if err != nil {
		t.Fatalf("ParseFalcoPayload(array): %v", err)
	}
	if len(alerts) != 2 {
		t.Fatalf("got %d alerts, want 2", len(alerts))
	}
}

// TestParseFalcoPayloadMalformed verifies malformed/empty/contentless bodies error.
func TestParseFalcoPayloadMalformed(t *testing.T) {
	for _, body := range []string{
		"",
		"   ",
		"{not json",
		"[{\"rule\": \"x\"}",
		"42",
		`{"uuid":"x"}`, // neither rule nor output
		`[{"uuid":"x"}]`,
	} {
		if _, err := ParseFalcoPayload([]byte(body)); err == nil {
			t.Errorf("ParseFalcoPayload(%q) succeeded, want error", body)
		}
	}
}

// TestAppendIngested verifies ingested events land in the ring buffer newest
// first and respect the cap.
func TestAppendIngested(t *testing.T) {
	b := NewBuffer()
	for i := 0; i < maxEvents+10; i++ {
		e := FalcoAlert{
			UUID:   "id",
			Rule:   "Terminal shell in container",
			Output: "shell",
			Time:   time.Now().UTC().Format(time.RFC3339Nano),
		}.Normalize()
		b.Append(e)
	}
	if got := b.Len(); got != maxEvents {
		t.Fatalf("buffer length = %d, want %d", got, maxEvents)
	}
	for _, e := range b.List("", 50) {
		if e.Simulated {
			t.Fatal("ingested event stored with Simulated=true")
		}
	}
}
