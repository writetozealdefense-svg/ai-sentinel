// Package risk implements the deterministic risk score defined in
// docs/PHASE1-INTEGRATION.md (Interface 3). The score is a pure function of the
// detection stats so it can be unit-tested without any external dependency.
package risk

import "math"

// severityWeights is the per-category severity contribution. The score uses the
// maximum weight across the categories present in a scan.
var severityWeights = map[string]int{
	"SECRET": 40,
	"PHI":    35,
	"PCI":    35,
	"PII":    25,
	"CODE":   20,
	"OTHER":  5,
}

// Input is the set of detection-svc stats needed to compute a score.
type Input struct {
	Categories     []string
	TotalFindings  int
	HighestScore   float64
	Sanctioned     bool
}

// clamp constrains v to the inclusive range [lo, hi].
func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// SeverityWeight returns max over categories of the severity weight table.
// Unknown categories contribute 0.
func SeverityWeight(categories []string) int {
	max := 0
	for _, c := range categories {
		if w, ok := severityWeights[c]; ok && w > max {
			max = w
		}
	}
	return max
}

// Score computes the deterministic 0..100 risk score:
//
//	severityWeight = max over categories of {SECRET:40, PHI:35, PCI:35, PII:25, CODE:20, OTHER:5}
//	volume         = min(totalFindings, 20) * 2          # cap 40
//	confidence     = round(highestScore * 15)            # 0..15
//	unsanctioned   = sanctioned ? 0 : 10
//	score          = clamp(severityWeight + volume + confidence + unsanctioned, 0, 100)
func Score(in Input) int {
	severity := SeverityWeight(in.Categories)

	vol := in.TotalFindings
	if vol > 20 {
		vol = 20
	}
	if vol < 0 {
		vol = 0
	}
	volume := vol * 2

	confidence := int(math.Round(in.HighestScore * 15))

	unsanctioned := 10
	if in.Sanctioned {
		unsanctioned = 0
	}

	return clamp(severity+volume+confidence+unsanctioned, 0, 100)
}
