package risk

import "testing"

func TestScore(t *testing.T) {
	cases := []struct {
		name string
		in   Input
		want int
	}{
		{
			// Scenario 1: bulk PII+PCI dump to unsanctioned ChatGPT.
			// severity=max(PII:25,PCI:35)=35, volume=min(113,20)*2=40,
			// confidence=round(1.0*15)=15, unsanctioned=10 => 100 (clamped).
			name: "scenario-1 bulk PII+PCI unsanctioned",
			in:   Input{Categories: []string{"PII", "PCI"}, TotalFindings: 113, HighestScore: 1.0, Sanctioned: false},
			want: 100,
		},
		{
			// Single PII finding, sanctioned, mid confidence.
			// severity=25, volume=1*2=2, confidence=round(0.9*15)=14, unsanctioned=0 => 41.
			name: "single PII sanctioned",
			in:   Input{Categories: []string{"PII"}, TotalFindings: 1, HighestScore: 0.9, Sanctioned: true},
			want: 41,
		},
		{
			// Nothing detected.
			name: "no findings",
			in:   Input{Categories: nil, TotalFindings: 0, HighestScore: 0, Sanctioned: true},
			want: 0,
		},
		{
			// SECRET dominates severity; volume capped at 40.
			// severity=40, volume=min(50,20)*2=40, confidence=round(0.99*15)=15, unsanctioned=10 => 105 -> 100.
			name: "secret bulk unsanctioned clamps to 100",
			in:   Input{Categories: []string{"SECRET", "CODE"}, TotalFindings: 50, HighestScore: 0.99, Sanctioned: false},
			want: 100,
		},
		{
			// CODE only, low volume, sanctioned.
			// severity=20, volume=3*2=6, confidence=round(0.5*15)=8, unsanctioned=0 => 34.
			name: "code low volume sanctioned",
			in:   Input{Categories: []string{"CODE"}, TotalFindings: 3, HighestScore: 0.5, Sanctioned: true},
			want: 34,
		},
		{
			// OTHER only.
			// severity=5, volume=2*2=4, confidence=round(0.2*15)=3, unsanctioned=10 => 22.
			name: "other unsanctioned",
			in:   Input{Categories: []string{"OTHER"}, TotalFindings: 2, HighestScore: 0.2, Sanctioned: false},
			want: 22,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Score(tc.in)
			if got != tc.want {
				t.Fatalf("Score(%+v) = %d, want %d", tc.in, got, tc.want)
			}
		})
	}
}

// TestScenario1MapsToBlockClass asserts the Scenario-1 score is in the
// high/BLOCK-class band (>= 70), matching the integration doc expectation.
func TestScenario1MapsToBlockClass(t *testing.T) {
	got := Score(Input{Categories: []string{"PII", "PCI"}, TotalFindings: 113, HighestScore: 1.0, Sanctioned: false})
	if got != 100 {
		t.Fatalf("scenario-1 score = %d, want 100", got)
	}
	if got < 70 {
		t.Fatalf("scenario-1 score %d not in high/BLOCK-class band (>=70)", got)
	}
}

func TestSeverityWeight(t *testing.T) {
	if w := SeverityWeight([]string{"PII", "PCI"}); w != 35 {
		t.Fatalf("SeverityWeight(PII,PCI) = %d, want 35", w)
	}
	if w := SeverityWeight([]string{"unknown"}); w != 0 {
		t.Fatalf("SeverityWeight(unknown) = %d, want 0", w)
	}
}
