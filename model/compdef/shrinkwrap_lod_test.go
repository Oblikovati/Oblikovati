// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/occurrence"
)

// boxVolume reports the volume of an occurrence's placed range box (the LOD's extent).
func boxVolume(o *occurrence.Occurrence) float64 { return float64(o.RangeBox().Volume()) }

// findSubstitute returns the single IsSubstitute occurrence in the collection.
func findSubstitute(t *testing.T, occs *occurrence.Occurrences) *occurrence.Occurrence {
	t.Helper()
	var found *occurrence.Occurrence
	for _, o := range occs.All() {
		if o.IsSubstitute() {
			if found != nil {
				t.Fatalf("expected one substitute occurrence, found a second %q", o.Name())
			}
			found = o
		}
	}
	if found == nil {
		t.Fatal("no substitute occurrence found")
	}
	return found
}

// TestShrinkwrapToPartWholeEnvelope builds a single bounding-box LOD from a two-part
// assembly: parts at [0,1]³ and [3,4]×[0,1]² give an envelope box of [0,4]×[0,1]² = 4.
func TestShrinkwrapToPartWholeEnvelope(t *testing.T) {
	t.Parallel()
	sub := NewAssemblyComponentDefinition()
	sub.Place("a:1", partWithBlock(t, math.P3(0, 0, 0), math.P3(1, 1, 1)), math.Identity4())
	sub.Place("b:1", partWithBlock(t, math.P3(3, 0, 0), math.P3(4, 1, 1)), math.Identity4())

	lod, err := ShrinkwrapToPart(sub, feature.ShrinkwrapDefinition{EnvelopeStyle: feature.EnvelopeWhole})
	if err != nil {
		t.Fatalf("ShrinkwrapToPart: %v", err)
	}
	if got := float64(lod.RangeBox().Volume()); got < 3.999999 || got > 4.000001 {
		t.Errorf("LOD range-box volume = %g, want 4 (whole bounding box)", got)
	}
}

// TestSubstituteWithShrinkwrapRegistersLOD is the substitute-representation path: the
// source assembly occurrence is suppressed and a single IsSubstitute LOD occurrence
// takes its place, carrying the shrinkwrap envelope's extent.
func TestSubstituteWithShrinkwrapRegistersLOD(t *testing.T) {
	t.Parallel()
	sub := NewAssemblyComponentDefinition()
	sub.Place("a:1", partWithBlock(t, math.P3(0, 0, 0), math.P3(1, 1, 1)), math.Identity4())
	sub.Place("b:1", partWithBlock(t, math.P3(3, 0, 0), math.P3(4, 1, 1)), math.Identity4())

	top := NewAssemblyComponentDefinition()
	subOcc := top.Place("sub:1", sub, math.Identity4())

	lodOcc, err := SubstituteWithShrinkwrap(top.Occurrences(), []*occurrence.Occurrence{subOcc},
		"sub-lod", sub, feature.ShrinkwrapDefinition{EnvelopeStyle: feature.EnvelopeWhole}, math.Identity4())
	if err != nil {
		t.Fatalf("SubstituteWithShrinkwrap: %v", err)
	}
	if !subOcc.Suppressed() {
		t.Error("source occurrence was not suppressed by substitution")
	}
	if !lodOcc.IsSubstitute() {
		t.Error("LOD occurrence is not flagged IsSubstitute")
	}
	if got := boxVolume(findSubstitute(t, top.Occurrences())); got < 3.999999 || got > 4.000001 {
		t.Errorf("substitute LOD extent volume = %g, want 4 (whole envelope)", got)
	}
}
