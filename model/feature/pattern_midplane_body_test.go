// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// Mid-plane placement and suppression driven through a real recompute (Oblikovati#1889) — the
// element arithmetic is covered in pattern_midplane_test.go; these check the bodies that come out.

// TestMidPlaneRectPatternPlacesBodiesAboutTheSeed builds a 3-wide row and checks where the solids
// land. A one-way run reaches x=4; the mid-plane run of the same count reaches x=−2…+2 with the
// seed still at the origin, which is the whole point: the seed cannot move.
func TestMidPlaneRectPatternPlacesBodiesAboutTheSeed(t *testing.T) {
	t.Parallel()
	oneWay := patternCentresX(t, false)
	if want := []float64{0.5, 2.5, 4.5}; !floatsNear(oneWay, want) {
		t.Fatalf("one-way centres = %v, want %v", oneWay, want)
	}
	mid := patternCentresX(t, true)
	if want := []float64{-1.5, 0.5, 2.5}; !floatsNear(mid, want) {
		t.Fatalf("mid-plane centres = %v, want %v", mid, want)
	}
	// The seed's own body must be exactly where the source feature put it.
	if !containsNear(mid, 0.5) {
		t.Fatalf("the seed body left its place: centres %v have nothing at 0.5", mid)
	}
}

// TestSuppressedOccurrenceIsNotBuilt checks suppression reaches the recompute, not just the
// element list — the gap #1889 reports.
func TestSuppressedOccurrenceIsNotBuilt(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	src := NewBaseFeatures(fs).AddBase(prismBody())
	rect := NewPatternFeatures(fs).AddRectangular([]ID{src.ID()},
		func() int { return 4 }, func() int { return 1 }, math.V3(2, 0, 0), math.Vector3{})
	fs.Recompute()
	if n := len(fs.Result()); n != 4 {
		t.Fatalf("unsuppressed pattern = %d bodies, want 4", n)
	}
	if err := rect.SuppressElements([]int{1, 3}); err != nil {
		t.Fatalf("SuppressElements: %v", err)
	}
	fs.MarkDirty(fs.Item(1))
	fs.Recompute()
	if n := len(fs.Result()); n != 2 {
		t.Fatalf("suppressed pattern = %d bodies, want 2", n)
	}
	got := centresX(t, fs)
	if want := []float64{0.5, 4.5}; !floatsNear(got, want) {
		t.Fatalf("surviving centres = %v, want %v (occurrences 1 and 3 dropped)", got, want)
	}
}

// TestMidPlaneAndSuppressionRoundTrip keeps both across an .obk save/load, so a pattern reopens
// where it was drawn.
func TestMidPlaneAndSuppressionRoundTrip(t *testing.T) {
	t.Parallel()
	sk := sketch.NewSketches().Add(sketch.XYPlane())
	fs := NewPartFeatures(nil)
	src := NewExtrudeFeatures(fs).AddByDistanceExtent(sk, 0, ops.NewBody, func() float64 { return 5 })
	rect := NewPatternFeatures(fs).AddRectangular([]ID{src.ID()},
		func() int { return 5 }, func() int { return 1 }, math.V3(20, 0, 0), math.Vector3{})
	rect.Definition().MidPlaneX = true
	if err := rect.SuppressElements([]int{2, 4}); err != nil {
		t.Fatalf("SuppressElements: %v", err)
	}
	fs.Recompute()
	before := centresX(t, fs)

	data, err := fs.MarshalRecipe(oneSketch{sk})
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	fresh := NewPartFeatures(nil)
	if err := fresh.ApplyRecipe(data, oneSketch{sk}, nil); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	back := fresh.Item(1).Definition().(*RectangularPatternFeature)
	if !back.Definition().MidPlaneX {
		t.Error("midPlaneX did not survive the round-trip")
	}
	if got := back.SuppressedIndices(); !intsEqual(got, []int{2, 4}) {
		t.Errorf("restored suppression = %v, want [2 4]", got)
	}
	fresh.Recompute()
	if after := centresX(t, fresh); !floatsNear(after, before) {
		t.Errorf("reopened pattern sits at %v, want %v", after, before)
	}
}

// patternCentresX builds a 3-wide unit-cube row and returns the body centres, sorted along X.
func patternCentresX(t *testing.T, midPlane bool) []float64 {
	t.Helper()
	fs := NewPartFeatures(nil)
	src := NewBaseFeatures(fs).AddBase(prismBody()) // unit cube [0,1]^3, centre x = 0.5
	rect := NewPatternFeatures(fs).AddRectangular([]ID{src.ID()},
		func() int { return 3 }, func() int { return 1 }, math.V3(2, 0, 0), math.Vector3{})
	rect.Definition().MidPlaneX = midPlane
	fs.Recompute()
	return centresX(t, fs)
}

// centresX returns each result body's bounding-box centre X, ascending.
func centresX(t *testing.T, fs *PartFeatures) []float64 {
	t.Helper()
	out := make([]float64, 0, len(fs.Result()))
	for _, b := range fs.Result() {
		out = append(out, b.RangeBox().Center().X)
	}
	sortFloats(out)
	return out
}

func sortFloats(v []float64) {
	for i := 1; i < len(v); i++ {
		for j := i; j > 0 && v[j] < v[j-1]; j-- {
			v[j], v[j-1] = v[j-1], v[j]
		}
	}
}

func containsNear(v []float64, want float64) bool {
	for _, x := range v {
		if stdmath.Abs(x-want) < 1e-9 {
			return true
		}
	}
	return false
}
