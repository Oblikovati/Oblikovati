// SPDX-License-Identifier: GPL-2.0-only

package tessellate

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Regression for the large spherical-zone tessellation bug (OCCT blend/simple J2). An imported sphere
// zone that reaches an enclosed pole carries its parametric SEAM (a meridian down to the pole) in the
// outer loop, so sphereCapFan's newellUnit(outer3D) axis is biased by the seam+pole samples and the
// fan meshed the WRONG (small) cap — J2 tessellated the north cap (~4600) instead of the true zone
// (~26815), whole-body area 8525 vs ~30742. sphereZoneCapFan rebuilds the fan on the rim circle's own
// normal + the pole vertex, so the zone meshes to its analytic 2πR·h area. This fixture reproduces the
// seamed-zone face WITHOUT a STEP round-trip (a named fake), so the kernel guards the fix on its own.

// seamedZoneFace builds a sphere face whose outer loop is one full-circle rim (at height zc, kept side
// reaching the south pole) plus the two meridian seam arcs to that pole — the exact topology the STEP
// importer produces for a pole-reaching zone (unlike capBody's coplanar single-circle rim, which
// sphereCapFan already handles). Returns the sphere face alone (no closed solid: the smooth sphere has
// no real seam edge, so an explicit-seam body is non-manifold — the face is what the fix meshes).
func seamedZoneFace(t *testing.T, radius, zc float64) *topo.Face {
	t.Helper()
	sphere, err := geom.NewSphere(math.P3(0, 0, 0), radius)
	if err != nil {
		t.Fatalf("NewSphere(R=%.3f): %v", radius, err)
	}
	r0 := stdmath.Sqrt(radius*radius - zc*zc)
	q := math.P3(math.Scalar(r0), 0, math.Scalar(zc)) // rim seam vertex (+x meridian)
	p := math.P3(0, 0, math.Scalar(-radius))          // enclosed south pole (the fan apex)
	// RefDir points at q (azimuth 0) so the rim edge's param-0 point IS its seam vertex — as an
	// imported STEP edge always is; otherwise discretizeEdge would snap the endpoints off the ring.
	nrm, _ := math.UnitVector3FromVector(math.V3(0, 0, 1))
	ref, _ := math.UnitVector3FromVector(math.V3(1, 0, 0))
	rim := geom.Circle{Center: math.P3(0, 0, math.Scalar(zc)), Normal: nrm, RefDir: ref, Radius: r0}
	seam := meridianArc(t, radius, zc, p, q)
	lin := func(s string) topo.Lineage { return topo.NewLineage(topo.Tok("zone", s, 0)) }
	bld := topo.NewBuilder(true, lin("body"))
	vp, vq := bld.AddVertex(p, lin("vp")), bld.AddVertex(q, lin("vq"))
	er := bld.AddEdge(rim, vq, vq, lin("rim")) // full circle, start==end==Q
	ea := bld.AddEdge(seam, vp, vq, lin("seamA"))
	eb := bld.AddEdge(seam, vp, vq, lin("seamB")) // the other seam side, identical geometry
	bld.AddFace(sphere, lin("sph"), topo.OuterLoop(topo.Fwd(er), topo.Rev(ea), topo.Fwd(eb)))
	return bld.Build().Faces()[0]
}

// meridianArc builds the great-circle arc from the south pole p up to the rim seam vertex q, lying in
// the x-z plane (the sphere's y=0 seam meridian), via a mid-arc point below the equator.
func meridianArc(t *testing.T, radius, zc float64, p, q math.Point3) geom.Arc3d {
	t.Helper()
	midPolar := (stdmath.Pi + stdmath.Acos(zc/radius)) / 2 // between the south pole (π) and Q's latitude
	mid := math.P3(math.Scalar(radius*stdmath.Sin(midPolar)), 0, math.Scalar(radius*stdmath.Cos(midPolar)))
	arc, err := geom.Arc3dByThreePoints(p, mid, q)
	if err != nil {
		t.Fatalf("Arc3dByThreePoints(%v,%v,%v): %v", p, mid, q, err)
	}
	return arc
}

func TestSphereZoneReachingPoleMeshesWholeZone(t *testing.T) {
	t.Parallel()
	const R, zc = 50.0, 35.355339059327 // J2: rim at 45° latitude, kept zone reaches the south pole
	face := seamedZoneFace(t, R, zc)
	m := TessellateFace(face, PropertyQuality())

	// The decisive check: the LARGE zone (rim→south pole, 2πR·h) is meshed, not the small north cap
	// (~4600) capAxis's seam-biased newellUnit would have picked.
	h := R + zc // zone height, south pole (z=−R) to rim (z=zc)
	wantArea := 2 * stdmath.Pi * R * h
	if got := zoneMeshArea(m); stdmath.Abs(got-wantArea)/wantArea > 0.01 {
		t.Fatalf("zone meshed wrong region: area %.4f, want 2πRh=%.4f (rel %.4f > 1%%) — the J2 bug",
			got, wantArea, stdmath.Abs(got-wantArea)/wantArea)
	}
	// The fan must reach the enclosed south pole (its apex), not stop at the small cap.
	if minZ := zoneMeshMinZ(m); stdmath.Abs(minZ-(-R)) > 1e-3*R {
		t.Fatalf("fan did not reach the south pole: min z=%.4f, want %.4f", minZ, -R)
	}
	// Fold-free + positive volume: the sphere-center sector sum is positive and every facet winds
	// outward (its geometric normal agrees with the radial direction) — no inverted/degenerate facet.
	if v := sphereSectorVolume(m, math.P3(0, 0, 0)); v <= 0 {
		t.Fatalf("meshed zone has non-positive sector volume %.4f (folded/inverted)", v)
	}
	if folds := zoneMeshFolds(m, math.P3(0, 0, 0)); folds > 0 {
		t.Fatalf("meshed zone has %d folded (inward-winding) facets, want 0", folds)
	}
}

// zoneMeshArea sums the triangle areas of a mesh.
func zoneMeshArea(m *Mesh) float64 {
	var area float64
	for ti := 0; ti < m.TriangleCount(); ti++ {
		a, b, c := TriVerts(m, ti)
		area += float64(a.VectorTo(b).Cross(a.VectorTo(c)).Length()) / 2
	}
	return area
}

// zoneMeshMinZ returns the lowest vertex z, so a fan that reaches the south pole reads z ≈ −R.
func zoneMeshMinZ(m *Mesh) float64 {
	minZ := stdmath.Inf(1)
	for _, p := range m.Positions {
		minZ = stdmath.Min(minZ, float64(p.Z))
	}
	return minZ
}

// sphereSectorVolume is the signed volume of the solid sector swept from center to the meshed surface
// (Σ signed tetra (center,a,b,c)) — positive for an outward-wound zone, the mesh's "positive volume".
func sphereSectorVolume(m *Mesh, center math.Point3) float64 {
	var v float64
	for ti := 0; ti < m.TriangleCount(); ti++ {
		a, b, c := TriVerts(m, ti)
		v += float64(center.VectorTo(a).Dot(center.VectorTo(b).Cross(center.VectorTo(c)))) / 6
	}
	return v
}

// zoneMeshFolds counts facets whose geometric normal opposes the outward radial direction at their
// centroid — an inverted/folded triangle on a sphere zone (the surface is strictly convex outward).
func zoneMeshFolds(m *Mesh, center math.Point3) int {
	folds := 0
	for ti := 0; ti < m.TriangleCount(); ti++ {
		a, b, c := TriVerts(m, ti)
		gn := a.VectorTo(b).Cross(a.VectorTo(c))
		cen := math.P3((a.X+b.X+c.X)/3, (a.Y+b.Y+c.Y)/3, (a.Z+b.Z+c.Z)/3)
		if gn.Dot(center.VectorTo(cen)) < 0 {
			folds++
		}
	}
	return folds
}
