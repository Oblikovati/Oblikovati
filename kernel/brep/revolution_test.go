// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// revVolume validates the body as a watertight solid and returns its tessellated volume.
func revVolume(t *testing.T, b *topo.Body) float64 {
	t.Helper()
	if r := ops.Validate(b); !r.Valid || !b.IsSolid() {
		t.Fatalf("revolution not a valid solid: %+v", r.Issues)
	}
	if open := ops.BoundaryEdges(b); len(open) != 0 {
		t.Fatalf("revolution has %d boundary edges, want 0 (watertight)", len(open))
	}
	return ops.BodyGeometryProperties(b, ops.DefaultQuality()).Volume
}

// TestRevolutionTubeIsAnalyticAnnulus revolves a rectangular annular meridian into a tube and
// asserts it is a valid watertight solid with the analytic annulus volume AND two true cylindrical
// walls (the bore + outer) — the surfaces thread/chamfer/fillet attach to (#129).
func TestRevolutionTubeIsAnalyticAnnulus(t *testing.T) {
	const rIn, rOut, h = 2.5, 6.0, 20.0
	mer := []math.Point2{math.P2(rIn, 0), math.P2(rOut, 0), math.P2(rOut, h), math.P2(rIn, h)}

	body, err := brep.SolidOfRevolution(math.P3(0, 0, 0), math.V3(0, 0, 1), mer, "tube")
	if err != nil || body == nil {
		t.Fatalf("SolidOfRevolution(tube) = %v, %v; want a body", body, err)
	}
	got := revVolume(t, body)
	want := stdmath.Pi * (rOut*rOut - rIn*rIn) * h
	if rel := stdmath.Abs(got-want) / want; rel > 0.03 {
		t.Errorf("tube volume = %.4f, want analytic %.4f (rel %.4f > 3%% faceting band)", got, want, rel)
	}

	cyls := 0
	for _, f := range body.Faces() {
		if _, ok := f.Geometry().(geom.Cylinder); ok {
			cyls++
		}
	}
	if cyls != 2 {
		t.Errorf("tube has %d analytic cylinder faces, want 2 (bore + outer wall)", cyls)
	}
}

// TestRevolutionDiscIsSolidCylinder revolves a rectangle touching the axis into a SOLID cylinder
// (the inner edge is on the axis ⇒ disk caps, one cylindrical wall).
func TestRevolutionDiscIsSolidCylinder(t *testing.T) {
	const r, h = 5.0, 8.0
	mer := []math.Point2{math.P2(0, 0), math.P2(r, 0), math.P2(r, h), math.P2(0, h)}

	body, err := brep.SolidOfRevolution(math.P3(0, 0, 0), math.V3(0, 0, 1), mer, "disc")
	if err != nil || body == nil {
		t.Fatalf("SolidOfRevolution(disc) = %v, %v; want a body", body, err)
	}
	got := revVolume(t, body)
	want := stdmath.Pi * r * r * h
	if rel := stdmath.Abs(got-want) / want; rel > 0.03 {
		t.Errorf("disc volume = %.4f, want analytic %.4f (rel %.4f > 3%% band)", got, want, rel)
	}
	if n := len(body.Faces()); n != 3 {
		t.Errorf("solid cylinder has %d faces, want 3 (wall + 2 disc caps)", n)
	}
}

// TestRevolutionFullConeApex revolves a right triangle touching the axis into a SOLID CONE: an
// apex (r=0), a base disk, and one analytic cone wall.
func TestRevolutionFullConeApex(t *testing.T) {
	const r, h = 4.0, 9.0
	mer := []math.Point2{math.P2(0, 0), math.P2(r, 0), math.P2(0, h)} // base, rim, apex on axis
	body, err := brep.SolidOfRevolution(math.P3(0, 0, 0), math.V3(0, 0, 1), mer, "cone")
	if err != nil || body == nil {
		t.Fatalf("SolidOfRevolution(cone) = %v, %v; want a body", body, err)
	}
	got := revVolume(t, body)
	want := stdmath.Pi * r * r * h / 3
	if rel := stdmath.Abs(got-want) / want; rel > 0.03 {
		t.Errorf("cone volume = %.4f, want analytic %.4f (rel %.4f > 3%% band)", got, want, rel)
	}
	if n := coneFaceCount(body); n != 1 {
		t.Errorf("solid cone has %d analytic cone faces, want 1", n)
	}
}

// TestRevolutionChamferedCylinder revolves a cylinder profile with a 45° bevel on the top rim — the
// analytic CONE FRUSTUM a true chamfer of a circular edge produces (#127). It must be a valid solid
// with one cone face, one cylinder wall, and the expected (cylinder − corner-wedge-of-revolution)
// volume.
func TestRevolutionChamferedCylinder(t *testing.T) {
	const r, h, d = 5.0, 10.0, 2.0
	mer := []math.Point2{math.P2(0, 0), math.P2(r, 0), math.P2(r, h-d), math.P2(r-d, h), math.P2(0, h)}
	body, err := brep.SolidOfRevolution(math.P3(0, 0, 0), math.V3(0, 0, 1), mer, "cham")
	if err != nil || body == nil {
		t.Fatalf("SolidOfRevolution(chamfered) = %v, %v; want a body", body, err)
	}
	got := revVolume(t, body)
	// Pappus: full cylinder minus the revolved corner triangle's removed solid.
	full := stdmath.Pi * r * r * h
	removed := stdmath.Pi*r*r*d - stdmath.Pi*d*(3*r*r-3*r*d+d*d)/3 // ∫ frustum
	want := full - removed
	if rel := stdmath.Abs(got-want) / want; rel > 0.03 {
		t.Errorf("chamfered cylinder volume = %.4f, want ≈%.4f (rel %.4f > 3%%)", got, want, rel)
	}
	if c, k := coneFaceCount(body), cylFaceCount(body); c != 1 || k != 1 {
		t.Errorf("chamfered cylinder has %d cone + %d cylinder faces, want 1 + 1", c, k)
	}
}

// TestRevolutionFilletedCylinder revolves a cylinder profile with a quarter-round on the top rim —
// the analytic TORUS a true fillet of a circular edge produces (#127). It must be a valid solid with
// one torus face, one cylinder wall, and the expected (cylinder − rounded-corner) volume.
func TestRevolutionFilletedCylinder(t *testing.T) {
	const r, h, f = 5.0, 10.0, 2.0
	// Meridian: base disk, wall up to h-f, ARC (about (r-f, h-f)) to (r-f, h), top disk, axis.
	center := math.P2(r-f, h-f)
	verts := []brep.RevolveVertex{
		{P: math.P2(0, 0)},
		{P: math.P2(r, 0)},
		{P: math.P2(r, h-f)},
		{P: math.P2(r-f, h), ArcCenter: &center},
		{P: math.P2(0, h)},
	}
	body, err := brep.SolidOfRevolutionMeridian(math.P3(0, 0, 0), math.V3(0, 0, 1), verts, "fil")
	if err != nil || body == nil {
		t.Fatalf("SolidOfRevolutionMeridian(filleted) = %v, %v; want a body", body, err)
	}
	got := revVolume(t, body)
	// Full cylinder minus the revolved rounded-corner sliver (the rim square minus its quarter-disc),
	// by Pappus: removed = 2π·(first moment of area about the axis). The sliver = the f×f corner
	// square at (r−f..r, h−f..h) minus the quarter-disc of radius f about (r−f, h−f).
	sqMoment := (r - f/2) * (f * f)                                       // square centroid-r × area
	qdMoment := ((r - f) + 4*f/(3*stdmath.Pi)) * (stdmath.Pi * f * f / 4) // quarter-disc moment
	removed := 2 * stdmath.Pi * (sqMoment - qdMoment)
	want := stdmath.Pi*r*r*h - removed
	if rel := stdmath.Abs(got-want) / want; rel > 0.03 {
		t.Errorf("filleted cylinder volume = %.4f, want ≈%.4f (rel %.4f > 3%%)", got, want, rel)
	}
	if torus, cyl := torusFaceCount(body), cylFaceCount(body); torus != 1 || cyl != 1 {
		t.Errorf("filleted cylinder has %d torus + %d cylinder faces, want 1 + 1", torus, cyl)
	}
}

func torusFaceCount(b *topo.Body) int {
	return surfFaceCount(b, func(g geom.Surface) bool { _, ok := g.(geom.Torus); return ok })
}
func coneFaceCount(b *topo.Body) int {
	return surfFaceCount(b, func(g geom.Surface) bool { _, ok := g.(geom.Cone); return ok })
}
func cylFaceCount(b *topo.Body) int {
	return surfFaceCount(b, func(g geom.Surface) bool { _, ok := g.(geom.Cylinder); return ok })
}
func surfFaceCount(b *topo.Body, pred func(geom.Surface) bool) int {
	n := 0
	for _, f := range b.Faces() {
		if pred(f.Geometry()) {
			n++
		}
	}
	return n
}
