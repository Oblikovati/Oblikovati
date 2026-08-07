// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"math"
	"testing"

	omath "oblikovati.org/math"
)

// Mid-plane patterns and per-element suppression (Oblikovati#1889).

// TestPatternIndexShiftSplitsAboutSeed pins the split rule: the other occurrences divide as evenly
// as the count allows, and an even count gives the extra to the step's own side.
func TestPatternIndexShiftSplitsAboutSeed(t *testing.T) {
	cases := []struct {
		name     string
		count    int
		midPlane bool
		want     int
	}{
		{"one-way is never shifted", 5, false, 0},
		{"odd count splits exactly", 5, true, 2},
		{"three straddles by one", 3, true, 1},
		{"even count keeps the extra on the +step side", 4, true, 1},
		{"six leaves two behind and three ahead", 6, true, 2},
		{"a single occurrence has nothing to straddle", 1, true, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := patternIndexShift(c.count, c.midPlane); got != c.want {
				t.Fatalf("patternIndexShift(%d, %v) = %d, want %d", c.count, c.midPlane, got, c.want)
			}
		})
	}
}

// TestRectMidPlaneKeepsSeedAtIdentity is the property the whole design rests on: the seed does not
// move. Element 0 must stay the identity in every direction combination, because the seed's
// material was placed by the source features before the pattern ran and cannot be re-placed.
func TestRectMidPlaneKeepsSeedAtIdentity(t *testing.T) {
	step := omath.Vector3{X: 2}
	for _, mid := range []bool{false, true} {
		for _, n := range []int{1, 2, 3, 4, 5} {
			xf := rectTransforms(n, 1, step, omath.Vector3{}, patternIndexShift(n, mid), 0)
			if got := xf[0]; !matrixNear(got, omath.Identity4()) {
				t.Fatalf("midPlane=%v count=%d: element 0 moved to %v, want identity", mid, n, got)
			}
		}
	}
}

// TestRectMidPlaneStraddlesTheSeed checks the placements themselves, not just the shift: a
// three-wide mid-plane run must sit at −2, 0, +2 rather than 0, +2, +4.
func TestRectMidPlaneStraddlesTheSeed(t *testing.T) {
	step := omath.Vector3{X: 2}
	oneWay := offsetsX(t, rectTransforms(3, 1, step, omath.Vector3{}, patternIndexShift(3, false), 0))
	if want := []float64{0, 2, 4}; !floatsNear(oneWay, want) {
		t.Fatalf("one-way offsets = %v, want %v", oneWay, want)
	}
	mid := offsetsX(t, rectTransforms(3, 1, step, omath.Vector3{}, patternIndexShift(3, true), 0))
	// Element 0 is the seed, so the list reads seed-first: 0, then the remaining cells in grid
	// order (−2 then +2).
	if want := []float64{0, -2, 2}; !floatsNear(mid, want) {
		t.Fatalf("mid-plane offsets = %v, want %v", mid, want)
	}
}

// TestRectMidPlaneEvenCountFavoursTheStep pins Inventor's even-count tie-break: four occurrences
// cannot split evenly, and the extra goes on the side the step points at.
func TestRectMidPlaneEvenCountFavoursTheStep(t *testing.T) {
	forward := offsetsX(t, rectTransforms(4, 1, omath.Vector3{X: 2}, omath.Vector3{}, patternIndexShift(4, true), 0))
	if want := []float64{0, -2, 2, 4}; !floatsNear(forward, want) {
		t.Fatalf("forward-step offsets = %v, want %v (two ahead, one behind)", forward, want)
	}
	// Reversing the step is how an author moves the extra to the other side — the same control
	// Inventor spells NaturalXDirection.
	reversed := offsetsX(t, rectTransforms(4, 1, omath.Vector3{X: -2}, omath.Vector3{}, patternIndexShift(4, true), 0))
	if want := []float64{0, 2, -2, -4}; !floatsNear(reversed, want) {
		t.Fatalf("reversed-step offsets = %v, want %v", reversed, want)
	}
}

// TestRectMidPlaneGridStraddlesBothDirections checks the two directions are independent: a
// mid-plane X with a one-way Y must straddle in X only.
func TestRectMidPlaneGridStraddlesBothDirections(t *testing.T) {
	xf := rectTransforms(3, 2, omath.Vector3{X: 2}, omath.Vector3{Y: 5},
		patternIndexShift(3, true), patternIndexShift(2, false))
	if len(xf) != 6 {
		t.Fatalf("got %d occurrences, want 6", len(xf))
	}
	if !matrixNear(xf[0], omath.Identity4()) {
		t.Fatalf("seed moved: %v", xf[0])
	}
	var maxY, minY float64
	for _, m := range xf {
		p := m.TransformPoint(omath.Point3{})
		minY, maxY = math.Min(minY, p.Y), math.Max(maxY, p.Y)
		if p.X < -2-1e-9 || p.X > 2+1e-9 {
			t.Fatalf("X should straddle within ±2, got %g", p.X)
		}
	}
	if minY < -1e-9 || math.Abs(maxY-5) > 1e-9 {
		t.Fatalf("Y should run one way 0…5, got %g…%g", minY, maxY)
	}
}

// TestCircMidPlaneStraddlesTheSeed is the circular equivalent: a 90° three-up mid-plane array sits
// at −45°, 0, +45° about the seed.
func TestCircMidPlaneStraddlesTheSeed(t *testing.T) {
	axis := omath.Vector3{Z: 1}
	xf, err := circTransforms(3, math.Pi/4, omath.Point3{}, axis, patternIndexShift(3, true))
	if err != nil {
		t.Fatalf("circTransforms: %v", err)
	}
	if !matrixNear(xf[0], omath.Identity4()) {
		t.Fatalf("seed rotated away from identity: %v", xf[0])
	}
	// A point on +X, rotated by each occurrence, must land at −45°, 0 and +45°.
	want := []float64{0, -math.Pi / 4, math.Pi / 4}
	for i, m := range xf {
		p := m.TransformPoint(omath.Point3{X: 1})
		if got := math.Atan2(p.Y, p.X); math.Abs(got-want[i]) > 1e-9 {
			t.Fatalf("occurrence %d at %.4f rad, want %.4f", i, got, want[i])
		}
	}
}

// TestSuppressElementsRefusesTheSeed is the honest-boundary test. Element 0 is the source features'
// own material; accepting the request and silently doing nothing would be the worse failure.
func TestSuppressElementsRefusesTheSeed(t *testing.T) {
	var p patternBase
	p.rebuild(4)
	if err := p.SuppressElements([]int{0}); err == nil {
		t.Fatal("suppressing the seed should be refused, not silently ignored")
	}
	if err := p.SuppressElements([]int{-1}); err == nil {
		t.Fatal("a negative element index should be refused")
	}
	if got := p.ActiveCount(); got != 4 {
		t.Fatalf("a refused request must change nothing, got ActiveCount %d, want 4", got)
	}
}

// TestSuppressElementsDropsOccurrences checks suppression reaches both the element list and the
// active count, and that a later call REPLACES the set rather than accumulating.
func TestSuppressElementsDropsOccurrences(t *testing.T) {
	var p patternBase
	p.rebuild(5)
	if err := p.SuppressElements([]int{2, 4}); err != nil {
		t.Fatalf("SuppressElements: %v", err)
	}
	if got := p.ActiveCount(); got != 3 {
		t.Fatalf("ActiveCount = %d, want 3", got)
	}
	if !p.skip(2) || !p.skip(4) || p.skip(1) || p.skip(3) {
		t.Fatalf("wrong occurrences skipped: %+v", p.Elements())
	}
	if got := p.SuppressedIndices(); !intsEqual(got, []int{2, 4}) {
		t.Fatalf("SuppressedIndices = %v, want [2 4]", got)
	}
	if err := p.SuppressElements([]int{1}); err != nil {
		t.Fatalf("SuppressElements: %v", err)
	}
	if got := p.SuppressedIndices(); !intsEqual(got, []int{1}) {
		t.Fatalf("a second call must replace the set, got %v, want [1]", got)
	}
}

// TestSuppressionSurvivesResize checks a dropped occurrence stays dropped when the count changes,
// which is why the set is keyed on index rather than rebuilt per recompute.
func TestSuppressionSurvivesResize(t *testing.T) {
	var p patternBase
	p.rebuild(3)
	if err := p.SuppressElements([]int{2}); err != nil {
		t.Fatalf("SuppressElements: %v", err)
	}
	p.rebuild(6) // the pattern grew
	if !p.Elements()[2].Suppressed {
		t.Fatal("occurrence 2 should still be suppressed after the count grew")
	}
	if got := p.ActiveCount(); got != 5 {
		t.Fatalf("ActiveCount = %d, want 5", got)
	}
}

// offsetsX reads the X translation out of each occurrence transform.
func offsetsX(t *testing.T, xf []omath.Matrix4) []float64 {
	t.Helper()
	out := make([]float64, len(xf))
	for i, m := range xf {
		out[i] = m.TransformPoint(omath.Point3{}).X
	}
	return out
}

func floatsNear(got, want []float64) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if math.Abs(got[i]-want[i]) > 1e-9 {
			return false
		}
	}
	return true
}

func matrixNear(a, b omath.Matrix4) bool {
	for _, p := range []omath.Point3{{}, {X: 1}, {Y: 1}, {Z: 1}} {
		pa, pb := a.TransformPoint(p), b.TransformPoint(p)
		if math.Abs(pa.X-pb.X) > 1e-9 || math.Abs(pa.Y-pb.Y) > 1e-9 || math.Abs(pa.Z-pb.Z) > 1e-9 {
			return false
		}
	}
	return true
}

func intsEqual(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
