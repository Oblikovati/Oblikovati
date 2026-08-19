// SPDX-License-Identifier: GPL-2.0-only

package occurrence

import (
	stdmath "math"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/math"
)

func unitX(t *testing.T) math.UnitVector3 { return mustUnit(t, 1, 0, 0) }
func unitY(t *testing.T) math.UnitVector3 { return mustUnit(t, 0, 1, 0) }
func unitZ(t *testing.T) math.UnitVector3 { return mustUnit(t, 0, 0, 1) }

func mustUnit(t *testing.T, x, y, z math.Scalar) math.UnitVector3 {
	t.Helper()
	u, err := math.NewUnitVector3(x, y, z)
	if err != nil {
		t.Fatalf("NewUnitVector3(%g,%g,%g): %v", x, y, z, err)
	}
	return u
}

func TestRectangularPatternGrid(t *testing.T) {
	arr := RectangularArrangement{Dir1: unitX(t), Spacing1: 1, Count1: 3, Dir2: unitY(t), Spacing2: 1, Count2: 2}
	p := NewOccurrencePattern(unitComponent(), math.Identity4(), arr)
	if p.Count() != 6 {
		t.Fatalf("count = %d, want 6 (3×2)", p.Count())
	}
	// Positions run column-fastest: element 0 at origin, element 2 at +2x, element 3 (row 1) at +1y.
	for _, c := range []struct {
		i    int
		want math.Point3
	}{{0, math.P3(0, 0, 0)}, {2, math.P3(2, 0, 0)}, {3, math.P3(0, 1, 0)}} {
		if got := p.Element(c.i).Transform().TransformPoint(math.P3(0, 0, 0)); got != c.want {
			t.Errorf("element %d at %v, want %v", c.i, got, c.want)
		}
	}
}

// TestPatternTracksCountEditsPreservingState is the PBI-121 acceptance: a count edit
// updates the element set while surviving elements keep their suppression/reposition.
func TestPatternTracksCountEditsPreservingState(t *testing.T) {
	x, y := unitX(t), unitY(t)
	p := NewOccurrencePattern(unitComponent(), math.Identity4(),
		RectangularArrangement{Dir1: x, Spacing1: 1, Count1: 3, Dir2: y, Spacing2: 1, Count2: 1})
	if p.Count() != 3 {
		t.Fatalf("count = %d, want 3", p.Count())
	}
	p.Element(1).SetSuppressed(true)
	off := math.Translation4(math.V3(0, 0, 9))
	p.Element(2).Reposition(off)

	p.SetArrangement(RectangularArrangement{Dir1: x, Spacing1: 1, Count1: 5, Dir2: y, Spacing2: 1, Count2: 1})
	if p.Count() != 5 {
		t.Fatalf("count after edit = %d, want 5", p.Count())
	}
	if !p.Element(1).Suppressed() {
		t.Error("element 1 lost its suppression across the count edit")
	}
	if !p.Element(2).Repositioned() || p.Element(2).Transform() != off {
		t.Error("element 2 lost its reposition override across the count edit")
	}
	if p.Element(4).Suppressed() || p.Element(4).Repositioned() {
		t.Error("new trailing element 4 should be fresh (unsuppressed, not repositioned)")
	}
}

func TestCircularPatternPlacesAroundAxis(t *testing.T) {
	p := NewOccurrencePattern(unitComponent(), math.Translation4(math.V3(1, 0, 0)),
		CircularArrangement{Origin: math.P3(0, 0, 0), Axis: unitZ(t), Step: stdmath.Pi / 2, Count: 4})
	if p.Count() != 4 {
		t.Fatalf("count = %d, want 4", p.Count())
	}
	// The seed sits at (1,0,0); a quarter turn about Z carries element 1 to ≈(0,1,0).
	got := p.Element(1).Transform().TransformPoint(math.P3(0, 0, 0))
	if !got.IsEqualTo(math.P3(0, 1, 0), 1e-9) {
		t.Errorf("element 1 ≈ %v, want ≈{0 1 0}", got)
	}
}

func TestSuppressedPatternElementLeavesNoBox(t *testing.T) {
	p := NewOccurrencePattern(unitComponent(), math.Identity4(),
		RectangularArrangement{Dir1: unitX(t), Spacing1: 100, Count1: 2, Dir2: unitY(t), Spacing2: 1, Count2: 1})
	if got := p.RangeBox().Max.X; got != 101 { // unit boxes at x∈[0,1] and x∈[100,101]
		t.Fatalf("full pattern box maxX = %g, want 101", got)
	}
	p.Element(1).SetSuppressed(true)
	if got := p.RangeBox().Max.X; got != 1 {
		t.Errorf("box maxX after suppressing the far element = %g, want 1", got)
	}
}

func TestFeatureArrangementFollowsSuppliedOffsets(t *testing.T) {
	offsets := []math.Matrix4{math.Identity4(), math.Translation4(math.V3(7, 0, 0))}
	p := NewOccurrencePattern(unitComponent(), math.Identity4(), FeatureArrangement{Offsets: offsets})
	if p.Count() != 2 {
		t.Fatalf("count = %d, want 2", p.Count())
	}
	if got := p.Element(1).Transform().TransformPoint(math.P3(0, 0, 0)); got != (math.P3(7, 0, 0)) {
		t.Errorf("feature-driven element 1 = %v, want {7 0 0}", got)
	}
}

// TestPatternSuppressionExcludesSeed the persistent pattern-level suppression (#1976) counts only
// the GENERATED elements: element 0 is the seed, never suppressed by the pattern, so a fully
// suppressed 4-up pattern still leaves the seed, and the seed cannot be suppressed or repositioned.
func TestPatternSuppressionExcludesSeed(t *testing.T) {
	p := NewOccurrencePattern(unitComponent(), math.Identity4(),
		CircularArrangement{Origin: math.P3(0, 0, 0), Axis: unitZ(t), Step: math.Scalar(stdmath.Pi / 2), Count: 4})
	if p.Suppression() != types.NoneSuppressed {
		t.Errorf("fresh pattern suppression = %v, want none", p.Suppression())
	}
	if err := p.SetElementSuppressed(1, true); err != nil {
		t.Fatalf("suppress element 1: %v", err)
	}
	if p.Suppression() != types.SomeElementsSuppressed {
		t.Errorf("one suppressed element = %v, want some", p.Suppression())
	}
	p.SetSuppressed(true)
	if p.Suppression() != types.AllElementsSuppressed {
		t.Errorf("all suppressed = %v, want all", p.Suppression())
	}
	if p.Element(0).Suppressed() {
		t.Error("SetSuppressed must not suppress the seed (element 0)")
	}
	if err := p.SetElementSuppressed(0, true); err == nil {
		t.Error("suppressing the seed (element 0) should be rejected")
	}
	if err := p.RepositionElement(0, math.Identity4()); err == nil {
		t.Error("repositioning the seed (element 0) should be rejected")
	}
	if err := p.SetElementSuppressed(4, true); err == nil {
		t.Error("an out-of-range element should be rejected")
	}
}
