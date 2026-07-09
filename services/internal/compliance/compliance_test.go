package compliance

import (
	"path/filepath"
	"testing"
)

// fixturesDir resolves the repo's fixtures/compliance directory relative to this
// test package (services/internal/compliance -> ../../../fixtures/compliance).
func fixturesDir(t *testing.T) string {
	t.Helper()
	dir, err := filepath.Abs(filepath.Join("..", "..", "..", "fixtures", "compliance"))
	if err != nil {
		t.Fatalf("resolve fixtures dir: %v", err)
	}
	return dir
}

func loadSet(t *testing.T) *Set {
	t.Helper()
	s, err := Load(fixturesDir(t))
	if err != nil {
		t.Fatalf("Load fixtures: %v", err)
	}
	if len(s.Frameworks()) == 0 {
		t.Fatal("no frameworks loaded from fixtures")
	}
	return s
}

// TestPIIMapsToSevenFrameworks asserts the Scenario-1 mapping rule: a PII finding
// produces exactly one matched control per regional + documented framework that
// covers PII, with control IDs sourced verbatim from the fixtures (including the
// TODO-VERIFY placeholders for regional frameworks). HIPAA (PHI-only) must NOT match.
func TestPIIMapsToSevenFrameworks(t *testing.T) {
	s := loadSet(t)
	matches := s.MatchFrameworks([]string{"PII"})

	got := map[string]Match{}
	for _, m := range matches {
		if _, dup := got[m.Framework]; dup {
			t.Fatalf("framework %s matched more than once (want exactly one control)", m.Framework)
		}
		got[m.Framework] = m
	}

	// Expected framework -> expected control ID, all sourced from the fixtures.
	// Documented frameworks select their highest-severity PII control; regional
	// frameworks carry the verbatim TODO-VERIFY placeholder (Brief R5).
	want := map[string]string{
		"GDPR":        "Art. 5(1)(f)", // tie at "high" with Art. 32; first-loaded wins deterministically
		"ISO27001":    "A.5.34",       // tie at "high"; first-loaded wins
		"NIST_AI_RMF": "GOVERN 1.1",   // highest severity (high) PII control
		"PDPL":        "TODO-VERIFY",
		"NCA_ECC":     "TODO-VERIFY",
		"SAMA_CSF":    "TODO-VERIFY",
		"DPDPA":       "TODO-VERIFY",
	}

	if len(got) != len(want) {
		t.Fatalf("PII matched %d frameworks %v, want %d %v", len(got), keys(got), len(want), wkeys(want))
	}

	for fw, wantID := range want {
		m, ok := got[fw]
		if !ok {
			t.Fatalf("expected framework %s to match for PII, but it did not", fw)
		}
		if m.Control.ControlID != wantID {
			t.Fatalf("%s: control ID = %q, want %q (must come from fixture)", fw, m.Control.ControlID, wantID)
		}
		if m.Category != "PII" {
			t.Fatalf("%s: matched category = %q, want PII", fw, m.Category)
		}
	}

	// HIPAA is PHI-only and must not match a PII-only finding.
	if _, ok := got["HIPAA"]; ok {
		t.Fatal("HIPAA matched for PII-only finding, but it maps only PHI")
	}
}

// TestRegionalControlsArePlaceholders confirms the TODO-VERIFY placeholders and
// review markers are preserved (never invented).
func TestRegionalControlsArePlaceholders(t *testing.T) {
	s := loadSet(t)
	for _, fw := range []string{"PDPL", "NCA_ECC", "SAMA_CSF", "DPDPA"} {
		f, ok := s.Framework(fw)
		if !ok {
			t.Fatalf("framework %s not loaded", fw)
		}
		for _, c := range f.Controls {
			if c.ControlID != "TODO-VERIFY" {
				t.Fatalf("%s control ID = %q, want TODO-VERIFY placeholder", fw, c.ControlID)
			}
			if !IsReview(c) {
				t.Fatalf("%s control %q lost its review marker", fw, c.ControlID)
			}
		}
	}
}

// TestPCIMatching spot-checks a second category (PCI) used in Scenario 1.
func TestPCIMatching(t *testing.T) {
	s := loadSet(t)
	matches := s.MatchFrameworks([]string{"PCI"})
	got := map[string]string{}
	for _, m := range matches {
		got[m.Framework] = m.Control.ControlID
	}
	// ISO27001 A.8.12 covers PCI; NIST GOVERN 1.1 covers PCI; SAMA/NCA cover PCI.
	for _, fw := range []string{"ISO27001", "NIST_AI_RMF", "SAMA_CSF", "NCA_ECC"} {
		if _, ok := got[fw]; !ok {
			t.Fatalf("expected %s to match for PCI", fw)
		}
	}
	// GDPR has no PCI control -> must not match.
	if _, ok := got["GDPR"]; ok {
		t.Fatal("GDPR matched for PCI-only finding but maps no PCI control")
	}
}

func keys(m map[string]Match) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func wkeys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
