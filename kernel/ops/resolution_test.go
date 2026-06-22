// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"math"
	"testing"

	"oblikovati.org/kernel/subd"
	"oblikovati.org/kernel/topo"
	gmath "oblikovati.org/math"
)

// TestResolutionForSizeFloorsDegenerate covers the positive-resolution guarantee:
// zero, negative and NaN sizes all floor to minModelSize so a degenerate operand
// still has a strictly positive resolution (ADR-0042 §Phase 1).
func TestResolutionForSizeFloorsDegenerate(t *testing.T) {
	for _, size := range []float64{0, -5, 0.001, math.NaN()} {
		got := ResolutionForSize(size).Size()
		if got != minModelSize {
			t.Errorf("ResolutionForSize(%v).Size() = %v, want floor %v", size, got, minModelSize)
		}
	}
	if got := ResolutionForSize(50).Size(); got != 50 {
		t.Errorf("ResolutionForSize(50).Size() = %v, want 50", got)
	}
}

// TestResolutionScalesLinearlyWithSize is the core property: every length
// tolerance scales linearly with the model size, and the volume tolerance with
// size³. A 1000× larger model gets 1000× looser length tolerances — this is what
// makes the kernel scale-faithful instead of cm-anchored.
func TestResolutionScalesLinearlyWithSize(t *testing.T) {
	small := ResolutionForSize(1)
	big := ResolutionForSize(1000)
	const k = 1000.0

	lengths := []struct {
		name string
		of   func(Resolution) float64
	}{
		{"Weld", Resolution.Weld},
		{"Plane", Resolution.Plane},
		{"Grid", Resolution.Grid},
		{"Sew", Resolution.Sew},
	}
	for _, l := range lengths {
		if got, want := l.of(big), l.of(small)*k; !approxRel(got, want) {
			t.Errorf("%s: big=%v, want small×%g=%v", l.name, got, k, want)
		}
	}
	if got, want := big.Volume(), small.Volume()*k*k*k; !approxRel(got, want) {
		t.Errorf("Volume: big=%v, want small×%g³=%v", got, k, want)
	}
}

// TestResolutionReproducesHistoricalConstants pins the calibration: at a 1 cm
// reference part (size = 1) each derived tolerance equals the absolute constant it
// replaces in #1243, so the migration is behaviour-preserving at typical scale.
func TestResolutionReproducesHistoricalConstants(t *testing.T) {
	r := ResolutionForSize(1)
	cases := []struct {
		name string
		got  float64
		want float64 // the historical cm/cm³ constant
	}{
		{"Weld (arrWeld/weldPointTol)", r.Weld(), 1e-9},
		{"Plane (csgEps)", r.Plane(), 1e-7},
		{"Grid (weldGrid/onLineTol)", r.Grid(), 1e-6},
		{"Sew (defaultSewTolerance)", r.Sew(), 1e-4},
		{"Volume (boolean tol)", r.Volume(), 1e-6},
	}
	for _, c := range cases {
		if !approxRel(c.got, c.want) {
			t.Errorf("%s = %v, want historical %v", c.name, c.got, c.want)
		}
	}
}

// TestResolutionForPoints derives the size from a point set's bbox diagonal and
// floors an empty/degenerate set.
func TestResolutionForPoints(t *testing.T) {
	if got := ResolutionForPoints(nil).Size(); got != minModelSize {
		t.Errorf("ResolutionForPoints(nil).Size() = %v, want floor %v", got, minModelSize)
	}
	// A 3-4-5... actually a unit cube: diagonal = sqrt(3).
	cube := []gmath.Point3{
		gmath.P3(0, 0, 0), gmath.P3(2, 0, 0), gmath.P3(2, 2, 0), gmath.P3(0, 2, 0),
		gmath.P3(0, 0, 2), gmath.P3(2, 0, 2), gmath.P3(2, 2, 2), gmath.P3(0, 2, 2),
	}
	want := math.Sqrt(2*2 + 2*2 + 2*2) // 2-unit cube diagonal
	if got := ResolutionForPoints(cube).Size(); !approxRel(got, want) {
		t.Errorf("ResolutionForPoints(cube).Size() = %v, want %v", got, want)
	}
}

// TestResolutionForBody covers the body entry point: nil floors, and a populated
// body derives its size from the true RangeBox diagonal.
func TestResolutionForBody(t *testing.T) {
	if got := ResolutionForBody(nil).Size(); got != minModelSize {
		t.Errorf("ResolutionForBody(nil).Size() = %v, want floor %v", got, minModelSize)
	}
	if got := ResolutionForBody(&topo.Body{}).Size(); got != minModelSize {
		t.Errorf("ResolutionForBody(empty).Size() = %v, want floor %v", got, minModelSize)
	}
	box := subd.ToBody(subd.Box(3, 4, 12), "box") // 3-4-12 box: diagonal = 13
	if got := ResolutionForBody(box).Size(); !approxRel(got, 13) {
		t.Errorf("ResolutionForBody(3×4×12 box).Size() = %v, want 13", got)
	}
}

// approxRel reports whether got and want agree to a tight relative tolerance,
// independent of their magnitude (the values span 1e-9 to 1e-4).
func approxRel(got, want float64) bool {
	if want == 0 {
		return got == 0
	}
	return math.Abs(got-want)/math.Abs(want) < 1e-12
}
