// SPDX-License-Identifier: GPL-2.0-only

package cmdline

import "testing"

func TestDefaultVocabularyResolves(t *testing.T) {
	v := DefaultVocabulary()
	cases := map[string]string{
		"LINE":    "Sketch.Line",
		"l":       "Sketch.Line", // case-insensitive
		"E":       "Create.Extrude",
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
