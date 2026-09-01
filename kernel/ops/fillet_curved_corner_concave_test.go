// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// m5CornerOracle is M5's concave trihedral corner from OCCT tests/blend/simple, DRAWEXE-verified
// (corner-blend-weld-recon.md §5, sphere (45,14.49,45), 13 faces, area 61187.1). M5 is a box − r30
// cylindrical pocket; all three filleted edges are reentrant, so the rolling ball sits INSIDE the
// pocket void (radial R−r from the axis) with PER-FACE plane offsets — the single wall-ε foundation
// cannot place it. r = 5, wall R = 30, axis ∥ ẑ through (50,−10).
var (
	m5Vertex  = math.P3(50, 20, 50)
	m5Center  = math.P3(45, 14.494897427831781, 45) // OCCT corner-sphere centre (DRAWEXE)
	m5RadiusR = 30.0
	m5FilletR = 5.0
)

// TestM5ConcaveCornerSphere pins the per-face-sign concave corner-sphere solve against DRAWEXE: the
// centre is (45,14.49,45), the ball is r into the VOID of BOTH plane hosts (material-outward signed
// distance = +r, the concave sense — opposite the convex −r) and R−r from the cylinder axis (inside the
// pocket). Non-tautological: it asserts the OCCT coordinates and the exact per-face tangency senses, so
// a regression to the single-ε placement (which put the centre at (55,24.64,45), R+r outside) fails.
func TestM5ConcaveCornerSphere(t *testing.T) {
	t.Parallel()
	body := importCorpusSolid(t, "simple/M5")
	v := vertexNear(t, body, m5Vertex)
	if !cornerHasConcaveArm(v) {
		t.Fatalf("M5 corner gate: cornerHasConcaveArm=false, want true (a reentrant edge must open the per-face branch)")
	}
	faces := facesAtVertex(v)
	cyl, planes, ok := cylinderHostCorner(faces)
	if !ok {
		t.Fatalf("M5 corner is not [cylinder,plane,plane]")
	}
	cb, err := solveBlend(body, v, faces, m5FilletR)
	if err != nil {
		t.Fatalf("solveBlend declined the M5 concave corner: %v", err)
	}
	if got := float64(cb.center.DistanceTo(m5Center)); got > 1e-6 {
		t.Fatalf("M5 corner centre %v != DRAWEXE %v (residual %.3e > 1e-6)", cb.center, m5Center, got)
	}
	assertConcavePlaneSense(t, planes, cb.center, m5FilletR)
	if radial := cornerAxisDistance(cyl, cb.center); stdmath.Abs(radial-(m5RadiusR-m5FilletR)) > 1e-6 {
		t.Fatalf("M5 corner radial to axis = %.6f, want R−r = %.1f (ball inside the pocket void)", radial, m5RadiusR-m5FilletR)
	}
}

// assertConcavePlaneSense verifies the corner ball is exactly r into the VOID of both plane hosts —
// the material-outward signed distance is +r on EACH (the concave per-face sign), not the convex −r.
func assertConcavePlaneSense(t *testing.T, planes [2]*topo.Face, c math.Point3, r float64) {
	t.Helper()
	for i, f := range planes {
		pl := f.Geometry().(geom.Plane)
		signed := float64(pl.Origin.VectorTo(c).Dot(outwardPlaneNormal(f, pl)))
		if stdmath.Abs(signed-r) > 1e-6 {
			t.Fatalf("M5 plane %d signed distance = %+.6f, want +r = %.1f (ball r into the void, concave sense)", i, signed, r)
		}
	}
}

// cornerAxisDistance is the perpendicular distance from point c to the cylinder axis.
func cornerAxisDistance(cyl geom.Cylinder, c math.Point3) float64 {
	axis := cyl.AxisDir.AsVector()
	w := cyl.Origin.VectorTo(c)
	return float64(w.Sub(axis.Scale(w.Dot(axis))).Length())
}

// TestConvexCornerReductionUnchanged is the do-no-harm guard: B3's all-convex trihedral corner does NOT
// trip the concave gate (so it keeps the untouched single-ε path) AND the signed plane-pair line
// reduces byte-identically — planePairLineSigned(...,−1,−1,...) == planePairLine(...). The concave
// generalisation must not perturb any convex corner.
func TestConvexCornerReductionUnchanged(t *testing.T) {
	t.Parallel()
	body := importCorpusSolid(t, "simple/B3")
	v := vertexNear(t, body, math.P3(0, -50, 100))
	if cornerHasConcaveArm(v) {
		t.Fatalf("B3 corner gate: cornerHasConcaveArm=true, want false (an all-convex corner must skip the per-face branch)")
	}
	_, planes, ok := cylinderHostCorner(facesAtVertex(v))
	if !ok {
		t.Fatalf("B3 corner is not [cylinder,plane,plane]")
	}
	legacyP, legacyD, okL := planePairLine(planes, 10, v.Point())
	signedP, signedD, okS := planePairLineSigned(planes, 10, -1, -1, v.Point())
	if !okL || !okS || legacyP != signedP || legacyD != signedD {
		t.Fatalf("planePairLineSigned(−1,−1) not byte-identical to planePairLine: %v/%v vs %v/%v", signedP, signedD, legacyP, legacyD)
	}
}
