// SPDX-License-Identifier: GPL-2.0-only

package cmdline

import (
	"strings"
	"testing"
)

func TestDefaultVocabularyResolves(t *testing.T) {
	v := DefaultVocabulary()
	cases := map[string]string{
		"LINE":    "Sketch.Line",
		"line":    "Sketch.Line", // case-insensitive
		"EXT":     "Create.Extrude",
		"EXTRUDE": "Create.Extrude",
		"rec":     "Sketch.Rectangle",
		"UNDO":    "edit.undo",
	}
	for word, want := range cases {
		if got, ok := v.Resolve(word); !ok || got != want {
			t.Errorf("Resolve(%q) = %q,%v, want %q", word, got, ok, want)
		}
	}
}

func TestDefaultVocabularyUnknownWord(t *testing.T) {
	if _, ok := DefaultVocabulary().Resolve("FLERP"); ok {
		t.Error("unknown word should not resolve")
	}
}

// TestDefaultVocabularyNoDuplicateWords ensures the table has no word mapped twice;
// DefaultVocabulary panics on a duplicate, so building it is the assertion.
func TestDefaultVocabularyNoDuplicateWords(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("duplicate vocabulary word: %v", r)
		}
	}()
	_ = DefaultVocabulary()
}

func TestVocabularyMatches(t *testing.T) {
	v := DefaultVocabulary()
	// "LO" matches the LOFT and LOFTEDFLANGE commands, returned as their canonical names,
	// de-duped and sorted.
	got := v.Matches("lo")
	if len(got) < 2 || got[0] != "LOFT" {
		t.Fatalf("Matches(lo) = %v, want it to start with LOFT", got)
	}
	hasLoftedFlange := false
	for _, m := range got {
		if m == "LOFTEDFLANGE" {
			hasLoftedFlange = true
		}
		if m != strings.ToUpper(m) {
			t.Errorf("match %q is not a canonical upper-case word", m)
		}
	}
	if !hasLoftedFlange {
		t.Errorf("Matches(lo) = %v, want LOFTEDFLANGE included", got)
	}
	if sorted := append([]string(nil), got...); !slicesSorted(sorted) {
		t.Errorf("Matches not sorted: %v", got)
	}
	if v.Matches("") != nil {
		t.Error("empty prefix should return no matches")
	}
	if v.Matches("zzz") != nil {
		t.Error("a non-matching prefix should return no matches")
	}
}

func slicesSorted(s []string) bool {
	for i := 1; i < len(s); i++ {
		if s[i-1] > s[i] {
			return false
		}
	}
	return true
}

func TestVocabularyActionsDistinct(t *testing.T) {
	got := DefaultVocabulary().Actions()
	seen := map[string]bool{}
	for _, a := range got {
		if seen[a] {
			t.Errorf("Actions returned %q twice", a)
		}
		seen[a] = true
	}
	if len(got) == 0 {
		t.Error("Actions returned nothing")
	}
}
