// Package compliance loads the framework fixtures (fixtures/compliance/*.yaml)
// and maps finding categories to a single matched control per framework.
//
// Brief R5: control IDs come ONLY from the fixtures. This package never invents
// or hardcodes a control ID or clause number; regional frameworks legitimately
// carry "TODO-VERIFY" placeholders which are preserved verbatim.
package compliance

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Control is a single control mapping from a fixture file.
type Control struct {
	ControlID        string   `yaml:"controlId"`
	Title            string   `yaml:"title"`
	Description      string   `yaml:"description"`
	MappedCategories []string `yaml:"mappedCategories"`
	Severity         string   `yaml:"severity"`
	Review           string   `yaml:"review"`
}

// Framework is one loaded fixture file.
type Framework struct {
	Framework   string    `yaml:"framework"`
	DisplayName string    `yaml:"displayName"`
	Controls    []Control `yaml:"controls"`
}

// Set is the collection of all loaded frameworks, preserving file load order.
type Set struct {
	frameworks []Framework
	byName     map[string]Framework
}

// severityRank ranks severities for "highest-severity matching control" selection.
var severityRank = map[string]int{
	"low":      1,
	"medium":   2,
	"high":     3,
	"critical": 4,
}

// reviewMarkerPrefix is the verbatim marker the fixtures carry until a human verifies the control.
const reviewMarkerPrefix = "# REVIEW"

// Load reads every *.yaml in dir, parses it as a Framework, and returns a Set.
func Load(dir string) (*Set, error) {
	matches, err := filepath.Glob(filepath.Join(dir, "*.yaml"))
	if err != nil {
		return nil, fmt.Errorf("glob compliance fixtures: %w", err)
	}
	sort.Strings(matches)

	s := &Set{byName: map[string]Framework{}}
	for _, path := range matches {
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read fixture %s: %w", path, err)
		}
		var fw Framework
		if err := yaml.Unmarshal(raw, &fw); err != nil {
			return nil, fmt.Errorf("parse fixture %s: %w", path, err)
		}
		if fw.Framework == "" {
			// Skip non-framework yaml (e.g. stray files without a framework key).
			continue
		}
		s.frameworks = append(s.frameworks, fw)
		s.byName[fw.Framework] = fw
	}
	return s, nil
}

// Frameworks returns all loaded frameworks in deterministic (filename) order.
func (s *Set) Frameworks() []Framework { return s.frameworks }

// Framework looks up a single framework by its canonical name (e.g. "GDPR").
func (s *Set) Framework(name string) (Framework, bool) {
	fw, ok := s.byName[name]
	return fw, ok
}

// Match is one selected control for a framework given a set of categories.
type Match struct {
	Framework   string
	DisplayName string
	Control     Control
	Category    string // the matched category that selected this control
}

// IsReview reports whether a control still carries its review marker.
func IsReview(c Control) bool {
	return strings.HasPrefix(strings.TrimSpace(c.Review), reviewMarkerPrefix)
}

// categoryIntersect returns the first category in `categories` that the control
// maps, or "" if none. Iteration order of `categories` is honoured so callers
// get a deterministic matched category.
func categoryIntersect(c Control, categories []string) string {
	for _, want := range categories {
		for _, have := range c.MappedCategories {
			if want == have {
				return want
			}
		}
	}
	return ""
}

// MatchFrameworks selects, for each framework whose controls intersect the given
// categories, the single highest-severity matching control. Returns one Match
// per matching framework, in deterministic framework load order.
//
// This is the mapping rule from PHASE1-INTEGRATION.md Interface 4.
func (s *Set) MatchFrameworks(categories []string) []Match {
	var out []Match
	for _, fw := range s.frameworks {
		var best *Control
		var bestCat string
		for i := range fw.Controls {
			c := fw.Controls[i]
			cat := categoryIntersect(c, categories)
			if cat == "" {
				continue
			}
			if best == nil || severityRank[c.Severity] > severityRank[best.Severity] {
				cc := c
				best = &cc
				bestCat = cat
			}
		}
		if best != nil {
			out = append(out, Match{
				Framework:   fw.Framework,
				DisplayName: fw.DisplayName,
				Control:     *best,
				Category:    bestCat,
			})
		}
	}
	return out
}
