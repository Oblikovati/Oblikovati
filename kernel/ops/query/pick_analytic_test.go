// SPDX-License-Identifier: GPL-2.0-only

package query

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// hiQuality is a far finer tessellation than DefaultQuality; a mesh-based pick drifts between the
// two, an analytic ray∩surface pick does not. Used to prove Quality-independence (M48/C3). The
// rays below deliberately pierce facet INTERIORS (off-seam azimuths, off-axis), never a mesh
// vertex or seam edge, so a tessellated path genuinely drifts by the facet sagitta.
func hiQuality() Quality {
	return Quality{ChordTolerance: 1e-4, AngleTolerance: 0.2 * stdmath.Pi / 180}
}

// TestRayCastFaceHitLiesExactlyOnSphere fires an OFF-AXIS ray at an analytic sphere and asserts the
// hit lands ON the sphere (|hit − centre| = R) to a tight tolerance — a tessellated pick lands on a
// facet, inside the true surface by the facet sagitta.
func TestRayCastFaceHitLiesExactlyOnSphere(t *testing.T) {
	t.Parallel()
	const r = 3.0
	center := math.P3(0, 0, 0)
	body, err := brep.SolidSphere(center, r, "sphere")
	if err != nil {
		t.Fatalf("SolidSphere: %v", err)
	}
	origin := math.P3(0.7, 0.5, 10) // off the axis, off any pole/seam vertex
	dir := math.V3(0, 0, -1)
	_, dist, ok := RayCastFaces(body, origin, dir, DefaultQuality())
	if !ok {
		t.Fatal("off-axis ray missed the sphere")
	}
	hit := origin.TranslateBy(dir.Scale(math.Scalar(dist)))
	got := float64(center.DistanceTo(hit))
	if stdmath.Abs(got-r) > 1e-9 {
		t.Errorf("hit radius = %.12f, want exactly %v (analytic surface, not a facet)", got, r)
	}
}

// TestRayCastFacesQualityIndependent fires the SAME facet-interior ray at a cylinder side at two
// tessellation qualities and requires an identical face and hit distance, plus a hit exactly on the
// analytic radius. The old per-facet mesh path drifts with facet count; the analytic path is exact.
func TestRayCastFacesQualityIndependent(t *testing.T) {
	t.Parallel()
	const r, h = 3.0, 8.0
	body, err := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), r, h)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	// A ray along −X at y=1 crosses the side at azimuth ≈19.5° — between the 11.25° facet seams,
	// so it pierces a facet interior, not a seam edge.
	const y0 = 1.0
	xHit := stdmath.Sqrt(r*r - y0*y0)
	origin := math.P3(10, y0, h/2)
	dir := math.V3(-1, 0, 0)
	fLo, dLo, okLo := RayCastFaces(body, origin, dir, DefaultQuality())
	fHi, dHi, okHi := RayCastFaces(body, origin, dir, hiQuality())
	if !okLo || !okHi {
		t.Fatalf("ray missed the cylinder side (lo=%v hi=%v)", okLo, okHi)
	}
	if fLo != fHi {
		t.Errorf("hit face changed with quality: %p vs %p", fLo, fHi)
	}
	if stdmath.Abs(dLo-dHi) > 1e-9 {
		t.Errorf("hit distance drifted with quality: lo=%.12f hi=%.12f (Δ=%.2e)", dLo, dHi, dLo-dHi)
	}
	if stdmath.Abs(dLo-(10-xHit)) > 1e-9 {
		t.Errorf("hit distance = %.12f, want %.12f (analytic near side)", dLo, 10-xHit)
	}
	hit := origin.TranslateBy(dir.Scale(math.Scalar(dLo)))
	radial := stdmath.Hypot(float64(hit.X), float64(hit.Y))
	if stdmath.Abs(radial-r) > 1e-9 {
		t.Errorf("hit radial = %.12f, want %v (on the analytic side)", radial, r)
	}
}

// TestFindUsingRayFaceHitAnalytic requires FindUsingRay's face hit to land on the analytic surface
// and be Quality-independent, for a facet-interior ray.
func TestFindUsingRayFaceHitAnalytic(t *testing.T) {
	t.Parallel()
	const r, h = 2.0, 6.0
	body, err := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), r, h)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	const y0 = 0.7
	origin := math.P3(9, y0, h/2)
	dir := math.V3(-1, 0, 0)
	hitsLo := FindUsingRay(body, origin, dir, 0, DefaultQuality(), true)
	hitsHi := FindUsingRay(body, origin, dir, 0, hiQuality(), true)
	if len(hitsLo) == 0 || len(hitsHi) == 0 {
		t.Fatalf("no face hit (lo=%d hi=%d)", len(hitsLo), len(hitsHi))
	}
	if stdmath.Abs(hitsLo[0].Distance-hitsHi[0].Distance) > 1e-9 {
		t.Errorf("FindUsingRay distance drifted with quality: %.12f vs %.12f", hitsLo[0].Distance, hitsHi[0].Distance)
	}
	radial := stdmath.Hypot(float64(hitsLo[0].Point.X), float64(hitsLo[0].Point.Y))
	if stdmath.Abs(radial-r) > 1e-9 {
		t.Errorf("FindUsingRay hit radial = %.12f, want %v (analytic surface)", radial, r)
	}
}

// TestRayCastFacesGrazingBoundaryEdge fires a ray that travels exactly along a box's bottom-face
// plane so its pierce of the near side face lands precisely on their shared edge (a boundary point,
// not a trim interior). The strict even-odd interior test rejects a boundary point; the pick must
// still return that face — the mesh path counted it via inclusive triangle edges, and the rib
// to-next depth (Oblikovati#1882) depends on it. Regression for the analytic-pick boundary-grazing
// miss.
func TestRayCastFacesGrazingBoundaryEdge(t *testing.T) {
	t.Parallel()
	// Box spanning X[6,10], Y[0,4], Z[-2,2].
	body, err := brep.SolidBlock(math.P3(6, 0, -2), math.P3(10, 4, 2), "box")
	if err != nil {
		t.Fatalf("SolidBlock: %v", err)
	}
	// A +X ray at Y=0, Z=0: it lies in the Y=0 bottom-face plane and strikes the X=6 near face
	// exactly on its Y=0 edge (6,0,0).
	origin := math.P3(4, 0, 0)
	dir := math.V3(1, 0, 0)
	_, dist, ok := RayCastFaces(body, origin, dir, DefaultQuality())
	if !ok {
		t.Fatal("grazing ray missed the box (boundary pierce rejected)")
	}
	if stdmath.Abs(dist-2) > 1e-9 {
		t.Errorf("nearest hit t = %.12f, want 2 (the near face at X=6, not the far face at X=10)", dist)
	}
	// Quality-independent: the boundary here is a straight edge, discretized exactly.
	if _, dHi, okHi := RayCastFaces(body, origin, dir, hiQuality()); !okHi || stdmath.Abs(dHi-2) > 1e-9 {
		t.Errorf("grazing hit drifted/vanished at high quality: ok=%v t=%.12f", okHi, dHi)
	}
}

// TestLocateUsingPointFaceAnalytic requires LocateUsingPoint's nearest-face point to be the exact
// perpendicular foot on the analytic surface, independent of tessellation Quality. The query point
// is at an off-seam azimuth so the mesh foot would drift.
func TestLocateUsingPointFaceAnalytic(t *testing.T) {
	t.Parallel()
	const r, h = 4.0, 10.0
	body, err := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), r, h)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	// Outside the side at azimuth 20° (off the 11.25° seams), mid height. The nearest face point
	// is the radial foot on the side at radius r; the distance is (|p_xy| − r).
	const az = 20 * stdmath.Pi / 180
	const pr = 6.0
	p := math.P3(pr*stdmath.Cos(az), pr*stdmath.Sin(az), h/2)
	lo, okLo := LocateUsingPoint(body, topo.KindFace, p, 100, DefaultQuality())
	hi, okHi := LocateUsingPoint(body, topo.KindFace, p, 100, hiQuality())
	if !okLo || !okHi {
		t.Fatalf("no face located (lo=%v hi=%v)", okLo, okHi)
	}
	if stdmath.Abs(lo.Distance-hi.Distance) > 1e-9 {
		t.Errorf("nearest-face distance drifted with quality: %.12f vs %.12f", lo.Distance, hi.Distance)
	}
	if stdmath.Abs(lo.Distance-(pr-r)) > 1e-9 {
		t.Errorf("nearest-face distance = %.12f, want %v (analytic foot)", lo.Distance, pr-r)
	}
	radial := stdmath.Hypot(float64(lo.Point.X), float64(lo.Point.Y))
	if stdmath.Abs(radial-r) > 1e-9 {
		t.Errorf("nearest-face point radial = %.12f, want %v (on the analytic side)", radial, r)
	}
}
