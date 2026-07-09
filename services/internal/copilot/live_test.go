package copilot

import "testing"

// env builds a getenv func from a map (mirrors httpx.Env's getenv injection so
// tests never touch the real process environment).
func env(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

// TestLiveGateUnreachableByDefault: with COPILOT_LIVE unset the live client is
// nil, so the live code path is unreachable and mode is offline (Brief R3
// default).
func TestLiveGateUnreachableByDefault(t *testing.T) {
	if c := NewLiveClient(env(nil)); c != nil {
		t.Fatalf("live client must be nil with no env, got %+v", c)
	}
	if m := Mode(nil); m != "offline" {
		t.Fatalf("Mode(nil) = %q, want offline", m)
	}
}

// TestLiveGateRequiresBothFlagAndKey: COPILOT_LIVE=true alone (no key) and a
// key alone (no flag) both stay offline.
func TestLiveGateRequiresBothFlagAndKey(t *testing.T) {
	if c := NewLiveClient(env(map[string]string{"COPILOT_LIVE": "true"})); c != nil {
		t.Errorf("live client must be nil without COPILOT_API_KEY, got %+v", c)
	}
	if c := NewLiveClient(env(map[string]string{"COPILOT_API_KEY": "sk-test"})); c != nil {
		t.Errorf("live client must be nil without COPILOT_LIVE=true, got %+v", c)
	}
	if c := NewLiveClient(env(map[string]string{"COPILOT_LIVE": "false", "COPILOT_API_KEY": "sk-test"})); c != nil {
		t.Errorf("live client must be nil with COPILOT_LIVE=false, got %+v", c)
	}
}

// TestLiveGateEnabled: flag + key yields a live client with mode "live" and
// the default model claude-sonnet-5 unless COPILOT_MODEL overrides it.
func TestLiveGateEnabled(t *testing.T) {
	c := NewLiveClient(env(map[string]string{"COPILOT_LIVE": "true", "COPILOT_API_KEY": "sk-test"}))
	if c == nil {
		t.Fatal("live client must be non-nil with COPILOT_LIVE=true + COPILOT_API_KEY")
	}
	if m := Mode(c); m != "live" {
		t.Errorf("Mode(live) = %q, want live", m)
	}
	if c.Model() != "claude-sonnet-5" {
		t.Errorf("default model = %q, want claude-sonnet-5", c.Model())
	}
	c = NewLiveClient(env(map[string]string{"COPILOT_LIVE": "TRUE", "COPILOT_API_KEY": "sk-test", "COPILOT_MODEL": "claude-opus-4-8"}))
	if c == nil || c.Model() != "claude-opus-4-8" {
		t.Errorf("COPILOT_MODEL override not honoured: %+v", c)
	}
}
