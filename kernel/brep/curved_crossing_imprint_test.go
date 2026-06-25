// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Crossing-cylinder imprint (M2 Phase 2, Oblikovati/Oblikovati#1335). The imprint stage must trace the
// surface-surface intersection of two crossing cylinders as closed loops lying on BOTH surfaces — the
// boundary the split/stitch slices will build the watertight result on.

// onBothCylinders returns the largest distance any loop vertex sits off either cylinder surface.
func onBothCylinders(loop geom.Polyline, a, b geom.Cylinder) float64 {
	worst := 0.0
	for _, p := range loop.Vertices {
		ea := stdmath.Abs(float64(geom.SignedDistanceToSurface(a, p)))
		eb := stdmath.Abs(float64(geom.SignedDistanceToSurface(b, p)))
		worst = stdmath.Max(worst, stdmath.Max(ea, eb))
	}
	return worst
}

// TestCrossingCylinderImprintThinThroughFat traces a thin cylinder crossing a fat one perpendicularly:
// the rod's entry and exit through the fat wall give two clean closed loops, each on both surfaces.
func TestCrossingCylinderImprintThinThroughFat(t *testing.T) {
	fat, _ := SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 3, 12)    // axis z, R=3
	thin, _ := SolidCylinder(math.P3(-6, 0, 0), math.V3(1, 0, 0), 1.5, 12) // axis x, r=1.5, through the centre

	loops, ok := crossingCylinderImprint(fat, thin)
	if !ok || len(loops) != 2 {
		t.Fatalf("thin-through-fat imprint: ok=%v loops=%d, want 2 closed loops", ok, len(loops))
	}
	ca, _ := geom.NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3)
	cb, _ := geom.NewCylinder(math.P3(0, 0, 0), math.V3(1, 0, 0), 1.5)
	for i, lp := range loops {
		if !samePoint(lp.PointAt(0), lp.PointAt(1), geom.ResolutionForSize(1)) {
			t.Errorf("loop %d is not closed: %v vs %v", i, lp.PointAt(0), lp.PointAt(1))
		}
		if err := onBothCylinders(lp, ca, cb); err > 1e-5 {
			t.Errorf("loop %d sits %.2e off a cylinder surface, want it on both", i, err)
		}
	}
}

// TestCrossingCylinderImprintNonCylinderDefers: the imprint only handles bare cylinders.
func TestCrossingCylinderImprintNonCylinderDefers(t *testing.T) {
	block, _ := SolidBlock(math.P3(0, 0, 0), math.P3(2, 2, 2), "b")
	cyl, _ := SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 1, 4)
	if _, ok := crossingCylinderImprint(block, cyl); ok {
		t.Error("imprint of a block and a cylinder should defer (ok=false)")
	}
}

// TestCrossingCylinderImprintDisjointHasNoLoops: cylinders that do not meet trace no loop.
func TestCrossingCylinderImprintDisjointHasNoLoops(t *testing.T) {
	a, _ := SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 1, 4)
	b, _ := SolidCylinder(math.P3(20, 0, 0), math.V3(1, 0, 0), 1, 4) // far away
	if _, ok := crossingCylinderImprint(a, b); ok {
		t.Error("disjoint cylinders should trace no imprint loop (ok=false)")
	}
}
