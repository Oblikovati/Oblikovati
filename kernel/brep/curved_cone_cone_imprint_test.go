// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Cone–cone imprint (M2 Phase 2, Oblikovati/Oblikovati#1335). A tapered rod (a narrow frustum) crossing a
// fatter cone must trace as two clean closed loops (the rod's entry and exit), each lying on BOTH cone
// surfaces to tolerance — the SSI foundation the later boolean slices build on.

// TestConeConeImprintTwoCleanLoops crosses a narrow frustum (axis x, radius 0.8→1.5) through a fatter frustum
// (axis z, radius 2→4) and checks the trace is two closed loops, each on both surfaces.
func TestConeConeImprintTwoCleanLoops(t *testing.T) {
	fat, _ := SolidCylinderCone(math.P3(0, 0, -6), math.P3(0, 0, 6), 2, 4, "fat")
	thin, _ := SolidCylinderCone(math.P3(-6, 0, 0), math.P3(6, 0, 0), 0.8, 1.5, "thin")

	loops, ok := coneConeImprint(thin, fat)
	if !ok {
		t.Fatal("cone–cone imprint declined; want two crossing loops")
	}
	if n := len(loops); n != 2 {
		t.Fatalf("imprint traced %d loops, want 2 (the rod's entry and exit)", n)
	}
	ct, _, _, _ := coneSolidParams(facesOfAny(thin))
	cf, _, _, _ := coneSolidParams(facesOfAny(fat))
	if d := worstConeConeOffset(loops, ct, cf); d > 1e-4 {
		t.Errorf("imprint loops stray %.2e from the surfaces, want ≤ 1e-4 (must lie on both cones)", d)
	}
}

// TestConeConeImprintOrderIndependent: resolving works whichever cone is passed (traced) first.
func TestConeConeImprintOrderIndependent(t *testing.T) {
	fat, _ := SolidCylinderCone(math.P3(0, 0, -6), math.P3(0, 0, 6), 2, 4, "fat")
	thin, _ := SolidCylinderCone(math.P3(-6, 0, 0), math.P3(6, 0, 0), 0.8, 1.5, "thin")
	if _, ok := coneConeImprint(fat, thin); !ok {
		t.Error("cone–cone imprint should resolve with the fat cone traced first too")
	}
}

// TestConeConeImprintConeCylinderDefer: a cone and a cylinder are the cone–cylinder case, not this one.
func TestConeConeImprintConeCylinderDefer(t *testing.T) {
	cone, _ := SolidCylinderCone(math.P3(-6, 0, 0), math.P3(6, 0, 0), 1, 2.5, "cone")
	cyl, _ := SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 3, 12)
	if _, ok := coneConeImprint(cone, cyl); ok {
		t.Error("a cone and a cylinder should defer from the cone–cone imprint (ok=false)")
	}
}

// worstConeConeOffset is the largest distance of any loop vertex from either cone surface.
func worstConeConeOffset(loops []geom.Polyline, a, b geom.Cone) float64 {
	worst := 0.0
	for _, lp := range loops {
		for _, p := range lp.Vertices {
			worst = stdmath.Max(worst, stdmath.Max(distToConeSurface(a, p), distToConeSurface(b, p)))
		}
	}
	return worst
}
