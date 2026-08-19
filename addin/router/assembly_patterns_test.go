// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"fmt"
	stdmath "math"
	"testing"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
)

// Persistent occurrence-pattern editing over the wire (#1976). A pattern created by patternCreate
// is re-read, suppressed/unsuppressed as a whole or per element, repositioned, and deleted — the
// group the router used to throw away.

// makeCircularPattern creates a 4-up circular pattern of the seed and returns the session bits plus
// the created pattern.
func makeCircularPattern(t *testing.T) (*Router, *app.Session, wire.PatternInfo) {
	t.Helper()
	r, s, _, occs := assemblySessionWithBoxes(t, 0)
	var created wire.CreatePatternResult
	args := fmt.Sprintf(`{"seed":%d,"kind":"circular","origin":[0,0,0],"axis":[0,0,1],"angle":%g,"count":4}`,
		occs[0].ID(), stdmath.Pi/2)
	call(t, r, s, "assembly.patternCreate", args, &created)
	if created.Pattern.ID == 0 {
		t.Fatalf("patternCreate returned no pattern id: %+v", created.Pattern)
	}
	if len(created.Created) != 3 {
		t.Fatalf("pattern created %d occurrences, want 3 (4 total minus the seed)", len(created.Created))
	}
	return r, s, created.Pattern
}

// TestAssemblyPatternPersistsAndLists patternCreate records the pattern (id, kind, 4 elements), and
// patternList re-reads exactly it — the array is no longer thrown away.
func TestAssemblyPatternPersistsAndLists(t *testing.T) {
	r, s, pat := makeCircularPattern(t)
	if pat.Kind != "circular" || len(pat.Elements) != 4 || pat.Suppression != "none" {
		t.Errorf("pattern = %+v, want circular / 4 elements / none suppressed", pat)
	}
	var list wire.PatternListResult
	call(t, r, s, "assembly.patternList", `{}`, &list)
	if len(list.Patterns) != 1 || list.Patterns[0].ID != pat.ID {
		t.Fatalf("patternList = %+v, want the one pattern id %d", list.Patterns, pat.ID)
	}
}

// TestAssemblyPatternSuppressHidesAll suppressing the whole pattern reports "all" and hides every
// generated occurrence; unsuppressing restores "none" and reveals them.
func TestAssemblyPatternSuppressHidesAll(t *testing.T) {
	r, s, pat := makeCircularPattern(t)
	var info wire.PatternInfo
	call(t, r, s, "assembly.patternSetSuppressed", fmt.Sprintf(`{"pattern":%d,"suppressed":true}`, pat.ID), &info)
	if info.Suppression != "all" {
		t.Errorf("suppressed pattern reports %q, want all", info.Suppression)
	}
	if got := unsuppressedOccurrences(t, r, s); got != 1 {
		t.Errorf("a fully suppressed pattern leaves %d visible occurrences, want 1 (the seed)", got)
	}
	call(t, r, s, "assembly.patternSetSuppressed", fmt.Sprintf(`{"pattern":%d,"suppressed":false}`, pat.ID), &info)
	if info.Suppression != "none" {
		t.Errorf("unsuppressed pattern reports %q, want none", info.Suppression)
	}
	if got := unsuppressedOccurrences(t, r, s); got != 4 {
		t.Errorf("an unsuppressed 4-up pattern leaves %d visible occurrences, want 4", got)
	}
}

// TestAssemblyPatternElementSuppressReportsSome suppressing one element reports "some" at the
// pattern level and flags that element.
func TestAssemblyPatternElementSuppressReportsSome(t *testing.T) {
	r, s, pat := makeCircularPattern(t)
	var info wire.PatternInfo
	call(t, r, s, "assembly.patternElementSetSuppressed",
		fmt.Sprintf(`{"pattern":%d,"element":2,"suppressed":true}`, pat.ID), &info)
	if info.Suppression != "some" {
		t.Errorf("one suppressed element reports %q, want some", info.Suppression)
	}
	if !info.Elements[2].Suppressed {
		t.Errorf("element 2 is not flagged suppressed: %+v", info.Elements[2])
	}
	if got := unsuppressedOccurrences(t, r, s); got != 3 {
		t.Errorf("one suppressed element leaves %d visible occurrences, want 3", got)
	}
	// An out-of-range element is rejected, not silently ignored.
	if _, err := r.Handle(s, "assembly.patternElementSetSuppressed",
		[]byte(fmt.Sprintf(`{"pattern":%d,"element":99,"suppressed":true}`, pat.ID))); err == nil {
		t.Error("suppressing an out-of-range element should fail")
	}
}

// TestAssemblyPatternElementReposition repositioning one element flags it and moves its occurrence
// off the regular grid.
func TestAssemblyPatternElementReposition(t *testing.T) {
	r, s, pat := makeCircularPattern(t)
	var info wire.PatternInfo
	call(t, r, s, "assembly.patternElementReposition",
		fmt.Sprintf(`{"pattern":%d,"element":1,"transform":%s}`, pat.ID, transformJSON(9, 9, 0)), &info)
	if !info.Elements[1].Repositioned {
		t.Errorf("element 1 is not flagged repositioned: %+v", info.Elements[1])
	}
	if !occurrenceAt(t, r, s, 9, 9, 0) {
		t.Error("no occurrence sits at the repositioned (9,9,0); the reposition did not move an instance")
	}
}

// TestAssemblyPatternDelete deleting a pattern removes the occurrences it generated (the seed stays)
// and drops it from the list.
func TestAssemblyPatternDelete(t *testing.T) {
	r, s, pat := makeCircularPattern(t)
	var del wire.DeletePatternResult
	call(t, r, s, "assembly.patternDelete", fmt.Sprintf(`{"pattern":%d}`, pat.ID), &del)
	if del.Deleted != pat.ID {
		t.Errorf("delete returned %d, want %d", del.Deleted, pat.ID)
	}
	var tree wire.OccurrencesResult
	call(t, r, s, "assembly.occurrences", `{}`, &tree)
	if len(tree.Occurrences) != 1 {
		t.Errorf("after delete the tree has %d occurrences, want 1 (the seed)", len(tree.Occurrences))
	}
	var list wire.PatternListResult
	call(t, r, s, "assembly.patternList", `{}`, &list)
	if len(list.Patterns) != 0 {
		t.Errorf("a deleted pattern still lists: %+v", list.Patterns)
	}
	// Deleting an unknown pattern is an error.
	if _, err := r.Handle(s, "assembly.patternDelete", []byte(`{"pattern":99999}`)); err == nil {
		t.Error("deleting an unknown pattern should fail")
	}
}

// unsuppressedOccurrences counts the assembly's occurrences that are not suppressed.
func unsuppressedOccurrences(t *testing.T, r *Router, s *app.Session) int {
	t.Helper()
	var tree wire.OccurrencesResult
	call(t, r, s, "assembly.occurrences", `{}`, &tree)
	n := 0
	for _, o := range tree.Occurrences {
		if !o.Suppressed {
			n++
		}
	}
	return n
}

// occurrenceAt reports whether any occurrence sits at (x,y,z).
func occurrenceAt(t *testing.T, r *Router, s *app.Session, x, y, z float64) bool {
	t.Helper()
	var tree wire.OccurrencesResult
	call(t, r, s, "assembly.occurrences", `{}`, &tree)
	for _, o := range tree.Occurrences {
		c := o.Transform.Cells
		if c[3] == x && c[7] == y && c[11] == z {
			return true
		}
	}
	return false
}
