// SPDX-License-Identifier: GPL-2.0-only

package compdef_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/math"
	"oblikovati.org/model/occurrence"
)

// TestAssemblyPatternRoundTrips a persistent occurrence pattern — with one element suppressed and
// another repositioned off the grid — survives save/reopen: the pattern is re-recorded and re-linked
// to its (independently persisted) occurrences by name, and the per-element edits come back (#1976).
func TestAssemblyPatternRoundTrips(t *testing.T) {
	t.Parallel()
	store, ws, asm, widget, asmDef := placedAssembly(t)
	seed := placeFromFile(t, asm, widget, asmDef, "widget:1", math.Identity4())

	axis, _ := math.NewUnitVector3(0, 0, 1)
	arr := occurrence.CircularArrangement{Origin: math.P3(0, 0, 0), Axis: axis, Step: math.Scalar(stdmath.Pi / 2), Count: 4}
	pat := occurrence.NewOccurrencePattern(seed.Definition(), seed.Transform(), arr)
	generated := occurrence.PatternComponents(asmDef.Occurrences(), seed, pat)
	asmDef.Patterns().Add(pat, "Pattern1", seed, generated)
	if err := pat.SetElementSuppressed(2, true); err != nil {
		t.Fatalf("suppress element 2: %v", err)
	}
	if err := pat.RepositionElement(1, math.Translation4(math.V3(9, 9, 0))); err != nil {
		t.Fatalf("reposition element 1: %v", err)
	}

	def := reopenAssembly(t, store, ws, asm)
	if def.Patterns().Count() != 1 {
		t.Fatalf("reopened assembly has %d patterns, want 1", def.Patterns().Count())
	}
	rp := def.Patterns().Item(0)
	if rp.Name() != "Pattern1" || rp.Kind() != "circular" || rp.Count() != 4 {
		t.Errorf("restored pattern = {name:%q kind:%q count:%d}, want Pattern1/circular/4", rp.Name(), rp.Kind(), rp.Count())
	}
	if rp.Suppression() != types.SomeElementsSuppressed {
		t.Errorf("restored pattern suppression = %v, want some (element 2 suppressed)", rp.Suppression())
	}
	if !rp.Element(2).Suppressed() {
		t.Error("element 2 lost its suppression on reopen")
	}
	if !rp.Element(1).Repositioned() {
		t.Error("element 1 lost its reposition on reopen")
	}
	if got := rp.Element(1).Transform().Cells(); got != math.Translation4(math.V3(9, 9, 0)).Cells() {
		t.Errorf("element 1 override = %v, want the (9,9,0) translation", got)
	}
}
