// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	"math"
	"testing"

	gmath "oblikovati.org/math"
)

// TestResolutionForSizeFloorsDegenerate covers the positive-resolution guarantee: zero,
// negative and NaN sizes all floor to minModelSize so a degenerate operand still has a
// strictly positive resolution (ADR-0042 §Phase 1).
func TestResolutionForSizeFloorsDegenerate(t *testing.T) {
	for _, size := range []float64{0, -5, 0.001, math.NaN()} {
		if got := ResolutionForSize(size).Size(); got != minModelSize {
			t.Errorf("ResolutionForSize(%v).Size() = %v, want floor %v", size, got, minModelSize)
		}
	}
	if got := ResolutionForSize(50).Size(); got != 50 {
		t.Errorf("ResolutionForSize(50).Size() = %v, want 50", got)
	}
}

// TestResolutionScalesWithSize is the core property: every length tolerance scales
// linearly with the model size, and the volume tolerance with size³. A 1000× larger
// model gets 1000× looser length tolerances — this is what makes the kernel
// scale-faithful instead of cm-anchored.
func TestResolutionScalesWithSize(t *testing.T) {
	small, big := ResolutionForSize(1), ResolutionForSize(1000)
	const k = 1000.0
	for _, l := range []struct {
		name string
		of   func(Resolution) float64
	}{{"Weld", Resolution.Weld}, {"Plane", Resolution.Plane}, {"Sew", Resolution.Sew}} {
		if got, want := l.of(big), l.of(small)*k; !approxRel(got, want) {
			t.Errorf("%s: big=%v, want small×%g=%v", l.name, got, k, want)
		}
	}
	if got, want := big.Volume(), small.Volume()*k*k*k; !approxRel(got, want) {
		t.Errorf("Volume: big=%v, want small×%g³=%v", got, k, want)
	}
}

// TestResolutionCalibration pins each tolerance at a 1 cm reference part (size = 1). The
// vertex weld is the tight bounds-relative value (1e-9, the convex-hull precedent) rather
// than the old loose 1e-6 weldGrid — which only worked at ~cm scale and over-merged larger
// finely-detailed parts. The rest reproduce their historical constants.
func TestResolutionCalibration(t *testing.T) {
	r := ResolutionForSize(1)
	for _, c := range []struct {
		name string
		got  float64
		want float64
	}{
		{"Weld", r.Weld(), 1e-9},
		{"Plane", r.Plane(), 1e-7},
		{"Sew", r.Sew(), 1e-4},
		{"Volume", r.Volume(), 1e-6},
	} {
		if !approxRel(c.got, c.want) {
			t.Errorf("%s = %v, want %v", c.name, c.got, c.want)
		}
	}
}

// TestResolutionForBoxAndPoints covers the box / 3D / 2D constructors, including the
// empty/degenerate floors.
func TestResolutionForBoxAndPoints(t *testing.T) {
	if got := ResolutionForBox(gmath.EmptyBox()).Size(); got != minModelSize {
		t.Errorf("ResolutionForBox(empty).Size() = %v, want floor %v", got, minModelSize)
	}
	if got := ResolutionForPoints(nil).Size(); got != minModelSize {
		t.Errorf("ResolutionForPoints(nil).Size() = %v, want floor %v", got, minModelSize)
	}
	if got := ResolutionForPoints2D(nil).Size(); got != minModelSize {
		t.Errorf("ResolutionForPoints2D(nil).Size() = %v, want floor %v", got, minModelSize)
	}
	cube := []gmath.Point3{gmath.P3(0, 0, 0), gmath.P3(3, 0, 0), gmath.P3(3, 4, 12), gmath.P3(0, 0, 12)}
	if got := ResolutionForPoints(cube).Size(); !approxRel(got, 13) { // 3-4-12 box diagonal
		t.Errorf("ResolutionForPoints(3×4×12).Size() = %v, want 13", got)
	}
	sq := []gmath.Point2{gmath.P2(0, 0), gmath.P2(3, 0), gmath.P2(3, 4), gmath.P2(0, 4)}
	if got := ResolutionForPoints2D(sq).Size(); !approxRel(got, 5) { // 3-4-5 diagonal
		t.Errorf("ResolutionForPoints2D(3×4).Size() = %v, want 5", got)
	}
}

// approxRel reports whether got and want agree to a tight relative tolerance, independent
// of magnitude (the values span 1e-9 to 1e-4).
func approxRel(got, want float64) bool {
	if want == 0 {
		return got == 0
	}
	return math.Abs(got-want)/math.Abs(want) < 1e-12
}
