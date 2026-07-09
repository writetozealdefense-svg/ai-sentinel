// Falco ingest support (Phase 5 Helm hardening path). In RUNTIME_MODE=ingest
// the telemetry-generator stops generating synthetic events and instead accepts
// falcosidekick HTTP-webhook payloads on POST /runtime/events, normalizing them
// to the existing RuntimeEvent shape with Simulated=false — these are REAL
// runtime-security events from Falco.
package runtime

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// FalcoAlert is the JSON object falcosidekick's HTTP webhook output POSTs, one
// per Falco alert. Only the fields we normalize are declared; extra fields are
// ignored.
type FalcoAlert struct {
	UUID         string         `json:"uuid"`
	Output       string         `json:"output"`
	Priority     string         `json:"priority"`
	Rule         string         `json:"rule"`
	Time         string         `json:"time"`
	OutputFields map[string]any `json:"output_fields"`
	Source       string         `json:"source"`
	Tags         []string       `json:"tags"`
	Hostname     string         `json:"hostname"`
}

// MapFalcoPriority maps a Falco priority (case-insensitive) onto the
// RuntimeEvent priority enum (info | notice | warning | error | critical).
// Unknown values fall back to "info".
func MapFalcoPriority(p string) string {
	switch strings.ToLower(strings.TrimSpace(p)) {
	case "emergency", "alert", "critical":
		return "critical"
	case "error":
		return "error"
	case "warning":
		return "warning"
	case "notice":
		return "notice"
	case "informational", "info", "debug":
		return "info"
	default:
		return "info"
	}
}

// Normalize converts a falcosidekick alert into a RuntimeEvent with
// Simulated=false (this is a REAL event — Helm hardening path only).
//
// Mapping: eventId ← uuid (fallback: deterministic UUID from output+time),
// source ← "tetragon" only if the payload says so, else "falco" (sidekick's
// own source is usually "syscall", which is not part of the RuntimeEvent
// enum), rule ← rule, priority ← MapFalcoPriority, message ← output,
// namespace/pod/container ← output_fields (empty-safe), timestamp ← time
// (RFC3339; falls back to now if unparseable).
func (a FalcoAlert) Normalize() Event {
	id := a.UUID
	if id == "" {
		// Deterministic fallback so retried webhook deliveries map to the same id.
		id = uuid.NewSHA1(uuid.NameSpaceOID, []byte(a.Output+"|"+a.Time)).String()
	}

	source := "falco"
	if strings.EqualFold(strings.TrimSpace(a.Source), "tetragon") {
		source = "tetragon"
	}

	ts, err := time.Parse(time.RFC3339Nano, a.Time)
	if err != nil {
		ts = time.Now()
	}

	return Event{
		EventID:   id,
		Source:    source,
		Rule:      a.Rule,
		Priority:  MapFalcoPriority(a.Priority),
		Message:   a.Output,
		Namespace: a.stringField("k8s.ns.name"),
		Pod:       a.stringField("k8s.pod.name"),
		Container: a.stringField("container.name"),
		Simulated: false, // REAL Falco event (Helm path) — never true here
		Timestamp: ts.UTC(),
	}
}

// stringField returns output_fields[key] if it is a non-empty string.
func (a FalcoAlert) stringField(key string) string {
	if v, ok := a.OutputFields[key].(string); ok {
		return v
	}
	return ""
}

// ParseFalcoPayload decodes a falcosidekick webhook body. Sidekick POSTs a
// single JSON object per alert; a JSON array of alerts is accepted as a
// robustness bonus. Malformed JSON or alerts missing both rule and output
// return an error (the handler maps it to HTTP 400).
func ParseFalcoPayload(body []byte) ([]FalcoAlert, error) {
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		return nil, errors.New("empty body")
	}

	var alerts []FalcoAlert
	if trimmed[0] == '[' {
		if err := json.Unmarshal(trimmed, &alerts); err != nil {
			return nil, err
		}
	} else {
		var a FalcoAlert
		if err := json.Unmarshal(trimmed, &a); err != nil {
			return nil, err
		}
		alerts = []FalcoAlert{a}
	}

	for i, a := range alerts {
		if a.Rule == "" && a.Output == "" {
			return nil, fmt.Errorf("alert %d has neither rule nor output", i)
		}
	}
	return alerts, nil
}
