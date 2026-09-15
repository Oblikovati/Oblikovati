// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	"testing"

	"oblikovati.org/math"
)

// A WITNESS for each torus-section decline whose only fixture the chart reversal took away
// (Oblikovati/Oblikovati#3515, review round 4, Important-3).
//
// `616ba12a` moved three rows out of TestATorusPairOutsideTheEnvelopeIsRefusedByName and into the corpus
// that asserts geometry, because #3515 builds them exactly. That is the right move and it left two named
// declines — [DeclineTorusLaneStation] and [DeclineTorusSectionOffItsForm] — with no input anywhere in
// the tree that makes them fire. A refusal is a result, and a result nothing exercises is a result
// nothing protects: [DeclineTorusSectionOffItsForm] in particular is ADR-0066's own post-condition, so a
// change that broke torusSectionSatisfiesItsForm would have been caught by nothing.
//
// Both rows below are REAL inputs found by search, not planted curve sets, and both are written in exact
// decimals so the fixture survives being written down.

// TestAStationWithNoAzimuthDependenceIsRefusedByName is [DeclineTorusLaneStation]'s witness: a small
// tilted boss on the ring's outer equator whose station polynomial, at some tube angle, carries fewer
// than two extrema — so there is no branch PAIR to read there and the anchor is a seed from another
// station rather than a root of this one.
//
// ADR-0066's fixture for this name was "a torus boss sunk into the tube", which #3515 now builds exactly
// (TestATorusPairSectionLiesOnBothTori). This is a different boss, and it still cannot be read.
func TestAStationWithNoAzimuthDependenceIsRefusedByName(t *testing.T) {
	t.Parallel()
	ring := mustTorus(t, math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 1.5)
	boss := mustTorus(t, math.P3(6.5, 0, 0), math.V3(1, 1, 1), 2.5, 0.2)
	curves, why, ok := IntersectSurfacesAnalyticDeclining(ring, boss, ResolutionForSize(20))
	assertRefusedByName(t, curves, why, ok, DeclineTorusLaneStation)
}

// TestASectionOffTheFormIsRefusedByName is [DeclineTorusSectionOffItsForm]'s witness: a cone whose APEX
// sits on the ring's own tube surface.
//
// The apex is the quadric's singular locus, where ∇Q vanishes, so the quadric's first-order distance
// |Q|/|∇Q| — the length the post-condition gates on — climbs without bound near it. The reduction solves
// this pair happily: it returns two curves and DeclineNone, and every azimuth it reports is a certified
// root of its own station. What it cannot do is place those curves on the cone to within the modelling
// weld anywhere near the apex, and only a POSITION test sees that. Measured: the built section's worst
// first-order distance is 1.05e-04 against a weld of 6.50e-09, four orders over, at t = 0.4965.
//
// This is the composition failure the post-condition exists for, stated in ADR-0066: a root is certified
// where it is solved, a fold azimuth is an extremum rather than a root, a lane is labelled by an anchor
// carried from another station, and the azimuth census counts branches rather than placing them. Each is
// sound and their composition can still put a point off the surface.
func TestASectionOffTheFormIsRefusedByName(t *testing.T) {
	t.Parallel()
	ring := mustTorus(t, math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 1.5)
	cone, err := NewCone(ring.PointAt(0, 0), math.V3(1, 0, 1), 0.5)
	if err != nil {
		t.Fatalf("NewCone with its apex on the ring: %v", err)
	}
	curves, why, ok := TorusSection(ring, cone.QuadricForm(), ResolutionForSize(20))
	assertRefusedByName(t, curves, why, ok, DeclineTorusSectionOffItsForm)
	assertTheReductionItselfBuiltIt(t, ring, cone.QuadricForm())
}

// assertTheReductionItselfBuiltIt is what makes the row above a test of the POST-CONDITION rather than of
// whatever else might have refused the pair: the family reduction returns a section and no refusal, and
// it is the position gate alone that turns that into a decline.
func assertTheReductionItselfBuiltIt(t *testing.T, ring Torus, co TorusCoForm) {
	t.Helper()
	raw, why, ok := torusSectionOfFamily(ring, co, ResolutionForSize(20))
	if !ok || why != DeclineNone || len(raw) == 0 {
		t.Fatalf("the reduction refused %q with %d curves; this row no longer reaches the post-condition", why, len(raw))
	}
	if torusSectionSatisfiesItsForm(ring, co, raw) {
		t.Error("the post-condition accepts the section it is supposed to refuse")
	}
}

// assertRefusedByName requires the named refusal, no curves, and that the name reports as a conditioning
// demotion — which is what makes kernel/brep record it as a diag.Defect rather than swallow it.
func assertRefusedByName(t *testing.T, curves []Curve3, why SectionDecline, ok bool, want SectionDecline) {
	t.Helper()
	if ok || len(curves) != 0 {
		t.Fatalf("ok=%v with %d curves, want the named refusal %q", ok, len(curves), want)
	}
	if why != want {
		t.Fatalf("refused %q, want %q", why, want)
	}
	if !why.IsConditioning() {
		t.Error("a conditioning gate's refusal must report as one, so the boolean records a defect")
	}
}
