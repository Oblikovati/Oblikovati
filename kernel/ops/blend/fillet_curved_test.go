// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// b3TopRimEdge returns B3's top-rim CIRCLE edge — the Cyl∧Plane edge (R=50 wall ∧ z=100 cap)
// whose axis is ⊥ the plane, so classifyCurvedArm makes it a torus arm. Discriminated from the
// vertical wall LINE (also Cyl∧Plane, but one end at z=0) by both endpoints sitting on the cap.
func b3TopRimEdge(t *testing.T, body *topo.Body) *topo.Edge {
	t.Helper()
	for _, e := range body.Edges() {
		if _, _, ok := cylinderPlaneEdge(e); !ok {
			continue
		}
		if nearCapZ(e.StartVertex().Point()) && nearCapZ(e.EndVertex().Point()) {
			return e
		}
	}
	t.Fatalf("B3 top-rim Cyl∧Plane edge (both ends z≈100) not found")
	return nil
}

// nearCapZ reports whether p sits on B3's top cap (z ≈ 100); the fixtures are exact.
func nearCapZ(p math.Point3) bool { return stdmath.Abs(float64(p.Z)-100) < 1e-3 }

// TestComputeEdgeFillet_B3TorusArm drives the wired arm builder: computeEdgeFillet on B3's convex
// top-rim circle must now return an edgeFillet carrying the EXACT torus arm (OCCT BREP
// `5 0 0 90 … 40 10`) instead of the old curvedFilletError. The MajorRadius=40 (=R−r, not the
// concave R+r=60) is the convex/concave discrimination assertion: a flipped material side fails it.
func TestComputeEdgeFillet_B3TorusArm(t *testing.T) {
	t.Parallel()
	body := importCorpusSolid(t, "simple/B3")
	e := b3TopRimEdge(t, body)
	if ClassifyEdgeConvexity(e) != EdgeConvex {
		t.Fatalf("precondition: B3 top rim must be convex, got %s", ClassifyEdgeConvexity(e))
	}
	ef, err := computeEdgeFillet(body, filletPick{edge: e, r0: 10, r1: 10}, nil, nil, FillConcaveOutward)
	if err != nil {
		t.Fatalf("computeEdgeFillet on B3 curved rim errored (arm not built): %v", err)
	}
	tor, ok := ef.armSurface.(geom.Torus)
	if !ok {
		t.Fatalf("B3 curved rim arm = %T, want geom.Torus", ef.armSurface)
	}
	// Major = R−r = 40 (not the concave R+r = 60) is the convex/concave discrimination: a flipped
	// material side — an R−r torus on a concave rim, or R+r on this convex one — fails here.
	if stdmath.Abs(tor.MajorRadius-40) > 1e-6 || stdmath.Abs(tor.MinorRadius-10) > 1e-6 {
		t.Fatalf("B3 torus arm = {major %.6f, minor %.6f}, want {40,10} (convex R−r, not concave R+r=60)",
			tor.MajorRadius, tor.MinorRadius)
	}
}

// TestComputeEdgeFillet_VaryingCurvedFallsBack is the do-no-harm backstop: the exact arm surfaces are
// constant-radius primitives, so a VARYING pick on the same Cyl∧Plane edge must decline the arm and
// fall through to the honest curvedFilletError — never a wrong single-radius torus.
func TestComputeEdgeFillet_VaryingCurvedFallsBack(t *testing.T) {
	t.Parallel()
	body := importCorpusSolid(t, "simple/B3")
	e := b3TopRimEdge(t, body)
	if _, err := computeEdgeFillet(body, filletPick{edge: e, r0: 10, r1: 5}, nil, nil, FillConcaveOutward); err == nil {
		t.Fatalf("computeEdgeFillet accepted a varying-radius curved arm; want curvedFilletError fallback")
	}
}
