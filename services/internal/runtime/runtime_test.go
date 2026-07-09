package runtime

import "testing"

// TestGenerateAlwaysSimulated verifies every generated event carries
// Simulated=true (Brief R1 — no live kernel telemetry on the compose path).
func TestGenerateAlwaysSimulated(t *testing.T) {
	b := NewBuffer()
	for i := 0; i < 50; i++ {
		e := b.Generate()
		if !e.Simulated {
			t.Fatalf("event %d has Simulated=false; every event must be simulated", i)
		}
		if e.EventID == "" {
			t.Fatalf("event %d has empty eventId", i)
		}
		if e.Timestamp.Location().String() != "UTC" {
			t.Fatalf("event %d timestamp is not UTC: %v", i, e.Timestamp.Location())
		}
	}
}

// TestRingBufferCap verifies the buffer keeps at most maxEvents (200) events.
func TestRingBufferCap(t *testing.T) {
	b := NewBuffer()
	for i := 0; i < maxEvents+75; i++ {
		b.Generate()
	}
	if got := b.Len(); got != maxEvents {
		t.Fatalf("buffer length = %d, want %d", got, maxEvents)
	}
	// List with a large limit must also be capped at maxEvents.
	all := b.List("", 1000)
	if len(all) != maxEvents {
		t.Fatalf("List returned %d events, want %d", len(all), maxEvents)
	}
}

// TestListNewestFirst verifies List returns events newest-first.
func TestListNewestFirst(t *testing.T) {
	b := NewBuffer()
	for i := 0; i < 10; i++ {
		b.Generate()
	}
	got := b.List("", 10)
	if len(got) != 10 {
		t.Fatalf("List returned %d events, want 10", len(got))
	}
	for i := 1; i < len(got); i++ {
		if got[i].Timestamp.After(got[i-1].Timestamp) {
			t.Fatalf("events not newest-first: index %d (%v) is after index %d (%v)",
				i, got[i].Timestamp, i-1, got[i-1].Timestamp)
		}
	}
}

// TestListLimitAndSourceFilter verifies the limit default/cap and source filter.
func TestListLimitAndSourceFilter(t *testing.T) {
	b := NewBuffer()
	for i := 0; i < 20; i++ {
		b.Generate()
	}
	// Default limit (limit<=0 -> 50, but only 20 buffered).
	if got := len(b.List("", 0)); got != 20 {
		t.Fatalf("default-limit List returned %d, want 20", got)
	}
	// Explicit small limit.
	if got := len(b.List("", 5)); got != 5 {
		t.Fatalf("limit=5 List returned %d, want 5", got)
	}
	// Source filter: only falco events come back.
	for _, e := range b.List("falco", 200) {
		if e.Source != "falco" {
			t.Fatalf("source=falco filter returned a %q event", e.Source)
		}
	}
	for _, e := range b.List("tetragon", 200) {
		if e.Source != "tetragon" {
			t.Fatalf("source=tetragon filter returned a %q event", e.Source)
		}
	}
}
