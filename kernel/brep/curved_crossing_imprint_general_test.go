// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// General curved-crossing imprint (ADR-0058 phase 3). primaryCurvedSurface recognises a body's principal
// curved side; curvedImprintLoops traces the shared intersection loops of two such surfaces. These are the
// one entry the per-pair imprints (cone∩cone, cone∩cylinder, cylinder∩cylinder) now delegate to.

// TestPrimaryCurvedSurfaceRecognisesConeAndCylinder: a cone body yields its cone surface, a cylinder body
// its cylinder surface, and a planar block none.
func TestPrimaryCurvedSurfaceRecognisesConeAndCylinder(t *testing.T) {
	cone, _ := SolidCylinderCone(math.P3(0, 0, 0), math.P3(0, 0, 4), 1, 2, "cone")
	if s, ok := primaryCurvedSurface(cone); !ok {
		t.Error("a cone body must yield a primary curved surface")
	} else if _, isCone := s.(geom.Cone); !isCone {
		t.Errorf("a cone body's primary surface must be a geom.Cone, got %T", s)
	}

	cyl, _ := SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 2, 5)
	if s, ok := primaryCurvedSurface(cyl); !ok {
		t.Error("a cylinder body must yield a primary curved surface")
	} else if _, isCyl := s.(geom.Cylinder); !isCyl {
		t.Errorf("a cylinder body's primary surface must be a geom.Cylinder, got %T", s)
	}

	block, _ := SolidBlock(math.P3(0, 0, 0), math.P3(1, 1, 1), "block")
	if _, ok := primaryCurvedSurface(block); ok {
		t.Error("a planar block has no primary curved surface; want ok=false")
	}
}

// worstImprintOffset is the largest distance of any loop vertex from either operand surface.
func worstImprintOffset(loops []geom.Curve3, a, b geom.Surface) float64 {
	worst := 0.0
	for _, lp := range loops {
		for _, p := range imprintLoopPoints(lp) {
			da := stdmath.Abs(geom.SignedDistanceToSurface(a, p))
			db := stdmath.Abs(geom.SignedDistanceToSurface(b, p))
			worst = stdmath.Max(worst, stdmath.Max(da, db))
		}
	}
	return worst
}

// TestCurvedImprintLoopsTracesEveryCrossingPair: the one general imprint traces the two crossing loops for
// each ruled pair (cone∩cone, cone∩cylinder, cylinder∩cylinder), every loop vertex on BOTH surfaces.
func TestCurvedImprintLoopsTracesEveryCrossingPair(t *testing.T) {
	coneA, _ := SolidCylinderCone(math.P3(-6, 0, 0), math.P3(6, 0, 0), 0.8, 1.5, "thin")
	coneB, _ := SolidCylinderCone(math.P3(0, 0, -6), math.P3(0, 0, 6), 2, 4, "fat")
	coneC, _ := SolidCylinderCone(math.P3(-6, 0, 0), math.P3(6, 0, 0), 1, 2.5, "cone")
	cylA, _ := SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 3, 12)
	cylB, _ := SolidCylinder(math.P3(-6, 0, 0), math.V3(1, 0, 0), 1, 12)
	cylC, _ := SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 3, 12)

	cases := []struct {
		name string
		a, b *topo.Body
	}{
		{"cone∩cone", coneA, coneB},
		{"cone∩cylinder", coneC, cylA},
		{"cylinder∩cylinder", cylB, cylC},
	}
	for _, c := range cases {
		loops, ok := curvedImprintLoops(c.a, c.b, nil)
		if !ok || len(loops) != 2 {
			t.Fatalf("%s: curvedImprintLoops ok=%v loops=%d, want ok + 2 loops", c.name, ok, len(loops))
		}
		sa, _ := primaryCurvedSurface(c.a)
		sb, _ := primaryCurvedSurface(c.b)
		if d := worstImprintOffset(loops, sa, sb); d > 1e-4 {
			t.Errorf("%s: loops stray %.2e from the surfaces, want ≤ 1e-4 (must lie on both)", c.name, d)
		}
	}
}

// TestCurvedImprintLoopsDeclinesPlanarPair: two planar blocks have no curved side, so the general imprint
// declines (ok=false) rather than tracing.
func TestCurvedImprintLoopsDeclinesPlanarPair(t *testing.T) {
	x, _ := SolidBlock(math.P3(0, 0, 0), math.P3(2, 2, 2), "x")
	y, _ := SolidBlock(math.P3(1, 1, 1), math.P3(3, 3, 3), "y")
	if _, ok := curvedImprintLoops(x, y, nil); ok {
		t.Error("two planar blocks have no curved side; curvedImprintLoops must decline (ok=false)")
	}
}
