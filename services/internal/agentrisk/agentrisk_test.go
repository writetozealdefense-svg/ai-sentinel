package agentrisk

import (
	"reflect"
	"testing"
)

// scenario5 is the exact telemetry sequence from PHASE3-INTEGRATION.md Interface 3.
var scenario5 = []Event{
	{Action: ActionToolCall, Tool: "jira", Target: "PROJ-1042"},
	{Action: ActionToolCall, Tool: "github", Target: "payments-service"},
	{Action: ActionToolCall, Tool: "salesforce", Target: "accounts API"},
	{Action: ActionDataAccess, Tool: "salesforce", Target: "salesforce.accounts", Sensitive: true},
	{Action: ActionPrivilegeEscalation, Target: "github org admin"},
}

func TestScenario5Trajectory(t *testing.T) {
	want := []int{12, 24, 36, 51, 76}
	for i := range scenario5 {
		got, _ := Score(scenario5[:i+1])
		if got != want[i] {
			t.Errorf("after event %d: score = %d, want %d", i+1, got, want[i])
		}
	}
}

func TestDistinctToolCap(t *testing.T) {
	events := []Event{
		{Action: ActionToolCall, Tool: "a"},
		{Action: ActionToolCall, Tool: "b"},
		{Action: ActionToolCall, Tool: "c"},
		{Action: ActionToolCall, Tool: "d"},
		{Action: ActionToolCall, Tool: "e"},
	}
	got, _ := Score(events)
	if got != 36 {
		t.Errorf("5 distinct tools: score = %d, want 36 (cap)", got)
	}
	// Repeated use of the same tool does not add points.
	got, _ = Score([]Event{
		{Action: ActionToolCall, Tool: "jira"},
		{Action: ActionToolCall, Tool: "jira"},
		{Action: ActionToolCall, Tool: "jira"},
	})
	if got != 12 {
		t.Errorf("same tool 3x: score = %d, want 12", got)
	}
}

func TestSensitiveDataAccessCap(t *testing.T) {
	// Four DISTINCT sensitive targets: 4*15 = 60 -> capped at 30.
	events := []Event{
		{Action: ActionDataAccess, Target: "db1", Sensitive: true},
		{Action: ActionDataAccess, Target: "db2", Sensitive: true},
		{Action: ActionDataAccess, Target: "db3", Sensitive: true},
		{Action: ActionDataAccess, Target: "db4", Sensitive: true},
	}
	got, _ := Score(events)
	if got != 30 {
		t.Errorf("4 distinct sensitive data_access: score = %d, want 30 (cap)", got)
	}
	// Replaying the SAME sensitive access does not re-bill (idempotent re-runs).
	got, _ = Score([]Event{
		{Action: ActionDataAccess, Target: "db", Sensitive: true},
		{Action: ActionDataAccess, Target: "db", Sensitive: true},
	})
	if got != 15 {
		t.Errorf("same sensitive target twice: score = %d, want 15", got)
	}
	// Non-sensitive data_access adds nothing.
	got, _ = Score([]Event{{Action: ActionDataAccess, Target: "db", Sensitive: false}})
	if got != 0 {
		t.Errorf("non-sensitive data_access: score = %d, want 0", got)
	}
}

func TestPrivilegeEscalationAndExternalCallOnce(t *testing.T) {
	got, _ := Score([]Event{
		{Action: ActionPrivilegeEscalation},
		{Action: ActionPrivilegeEscalation},
	})
	if got != 25 {
		t.Errorf("privilege_escalation twice: score = %d, want 25 (once)", got)
	}
	got, _ = Score([]Event{
		{Action: ActionExternalCall},
		{Action: ActionExternalCall},
	})
	if got != 10 {
		t.Errorf("external_call twice: score = %d, want 10 (once)", got)
	}
}

func TestClampAt100(t *testing.T) {
	// Max everything: 36 + 30 + 25 + 10 = 101 -> clamp 100.
	events := []Event{
		{Action: ActionToolCall, Tool: "a"},
		{Action: ActionToolCall, Tool: "b"},
		{Action: ActionToolCall, Tool: "c"},
		{Action: ActionDataAccess, Target: "x", Sensitive: true},
		{Action: ActionDataAccess, Target: "y", Sensitive: true},
		{Action: ActionPrivilegeEscalation},
		{Action: ActionExternalCall},
	}
	got, _ := Score(events)
	if got != 100 {
		t.Errorf("maxed history: score = %d, want 100 (clamp)", got)
	}
}

func TestEmptyHistory(t *testing.T) {
	got, factors := Score(nil)
	if got != 0 || len(factors) != 0 {
		t.Errorf("empty history: score = %d factors = %v, want 0 and none", got, factors)
	}
}

func TestRiskFactors(t *testing.T) {
	_, factors := Score(scenario5)
	want := []string{
		"3 tools (jira, github, salesforce)",
		"sensitive data access (salesforce.accounts)",
		"privilege escalation",
	}
	if !reflect.DeepEqual(factors, want) {
		t.Errorf("factors = %v, want %v", factors, want)
	}
}

func TestCrossed(t *testing.T) {
	cases := []struct {
		prev, curr int
		want       bool
	}{
		{0, 12, false},   // below threshold
		{51, 76, true},   // the crossing (scenario-5 event 5)
		{69, 70, true},   // exact edge crossing
		{70, 70, false},  // already at threshold
		{76, 76, false},  // re-run after breach: no re-emit
		{76, 100, false}, // rising further after breach: no re-emit
		{0, 70, true},
		{0, 100, true},
	}
	for _, c := range cases {
		if got := Crossed(c.prev, c.curr); got != c.want {
			t.Errorf("Crossed(%d, %d) = %v, want %v", c.prev, c.curr, got, c.want)
		}
	}
}

// TestCrossingEmitsExactlyOnce replays scenario-5 event-by-event (then replays
// it again, as a re-run would) and counts threshold crossings: exactly one.
func TestCrossingEmitsExactlyOnce(t *testing.T) {
	var history []Event
	crossings := 0
	prev := 0
	replay := append(append([]Event{}, scenario5...), scenario5...) // run + re-run
	for _, e := range replay {
		history = append(history, e)
		curr, _ := Score(history)
		if Crossed(prev, curr) {
			crossings++
		}
		prev = curr
	}
	if crossings != 1 {
		t.Errorf("crossings = %d, want exactly 1", crossings)
	}
	// After the re-run the score stays at 76 (caps absorb the duplicates).
	final, _ := Score(history)
	if final != 76 {
		t.Errorf("score after re-run = %d, want 76", final)
	}
}
