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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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

// TestRevolutionSubMicronTaperIsCone pins audit A7 (#1603): a meridian wall with a REAL slope of
// 1e-8 rad (Δr = 1e-8 over 1 cm — under the old absolute revTol of 1e-7) must classify as an
// analytic geom.Cone, not silently flatten to a cylinder. Surface type is load-bearing identity
// downstream (recognizer, fillet eligibility, STEP), so a sub-µm taper the user modeled must
// survive classification.
func TestRevolutionSubMicronTaperIsCone(t *testing.T) {
	t.Parallel()
	// Wall kept thick (3 cm) so default-quality faceting error on the two walls cancels inside the
	// 3% band — a 0.5 cm-thin ring measures ~4.5% high at DefaultQuality (bore vs outer inscription
	// imbalance), a pre-existing tessellation-resolution effect unrelated to classification.
	const rIn, rOut, h, taper = 10.0, 13.0, 1.0, 1e-8 // inner wall slope = taper/h = 1e-8 rad
	mer := []math.Point2{math.P2(rIn, 0), math.P2(rOut, 0), math.P2(rOut, h), math.P2(rIn+taper, h)}
	body, err := brep.SolidOfRevolution(math.P3(0, 0, 0), math.V3(0, 0, 1), mer, "taper")
	if err != nil || body == nil {
		t.Fatalf("SolidOfRevolution(taper) = %v, %v; want a body", body, err)
	}
	got := revVolume(t, body)
	want := stdmath.Pi * (rOut*rOut - rIn*rIn) * h // taper's volume contribution is ~1e-7 — negligible
	if rel := stdmath.Abs(got-want) / want; rel > 0.03 {
		t.Errorf("tapered tube volume = %.4f, want ≈%.4f (rel %.4f > 3%% band)", got, want, rel)
	}
	if c, k := coneFaceCount(body), cylFaceCount(body); c != 1 || k != 1 {
		t.Errorf("tapered tube has %d cone + %d cylinder faces, want 1 cone (inner taper) + 1 cylinder (outer)", c, k)
	}
}

// TestRevolutionLargeRadiusCylinderStaysCylinder pins audit A7's other direction (#1603): a true
// axis-parallel wall at radial coordinate 1e4 carries upstream model→(r,z) projection noise far
// above the old absolute 1e-7 (here 3e-7 ≈ 3e-11 relative), yet its SLOPE is 3e-11 — a cylinder
// beyond doubt. The old absolute compare promoted it to a cone; classification must read the
// dimensionless slope instead.
func TestRevolutionLargeRadiusCylinderStaysCylinder(t *testing.T) {
	t.Parallel()
	const rIn, rOut, h, noise = 1e4, 1.05e4, 1e4, 3e-7
	mer := []math.Point2{math.P2(rIn, 0), math.P2(rOut, 0), math.P2(rOut, h), math.P2(rIn+noise, h)}
	body, err := brep.SolidOfRevolution(math.P3(0, 0, 0), math.V3(0, 0, 1), mer, "bigtube")
	if err != nil || body == nil {
		t.Fatalf("SolidOfRevolution(bigtube) = %v, %v; want a body", body, err)
	}
	got := revVolume(t, body)
	want := stdmath.Pi * (rOut*rOut - rIn*rIn) * h
	if rel := stdmath.Abs(got-want) / want; rel > 0.03 {
		t.Errorf("large tube volume = %.4g, want ≈%.4g (rel %.4f > 3%% band)", got, want, rel)
	}
	if c, k := coneFaceCount(body), cylFaceCount(body); c != 0 || k != 2 {
		t.Errorf("large tube has %d cone + %d cylinder faces, want 0 cones + 2 cylinders (bore + outer)", c, k)
	}
}

// TestRevolutionTaperClassIsScaleInvariant pins audit A7's core demand (#1603): the SAME
// normalized taper (slope 1e-6 rad) must classify identically — as a cone — at 1e-3×, 1× and
// 1e3× model scale. The old absolute revTol flipped the 1e-3× copy to a cylinder (Δr = 1e-9
// < 1e-7) while the 1× and 1e3× copies stayed cones.
func TestRevolutionTaperClassIsScaleInvariant(t *testing.T) {
	t.Parallel()
	const slope = 1e-6
	for _, scale := range []float64{1e-3, 1, 1e3} {
		rIn, rOut, h := 10*scale, 10.5*scale, 1*scale
		mer := []math.Point2{
			math.P2(rIn, 0), math.P2(rOut, 0), math.P2(rOut, h), math.P2(rIn+slope*h, h),
		}
		body, err := brep.SolidOfRevolution(math.P3(0, 0, 0), math.V3(0, 0, 1), mer, "sweep")
		if err != nil || body == nil {
			t.Fatalf("scale %g: SolidOfRevolution = %v, %v; want a body", scale, body, err)
		}
		if c, k := coneFaceCount(body), cylFaceCount(body); c != 1 || k != 1 {
			t.Errorf("scale %g: %d cone + %d cylinder faces, want 1 + 1 (classification must not depend on scale)",
				scale, c, k)
		}
	}
}

// TestRevolutionAxisWeldIsScaleRelative pins the on-axis vertex test's move to a meridian-relative
// tolerance (#1603): on a 1e4-extent part, an inner radius of 1e-4 (1e-8 of the model — far below
// the on-line classification resolution) is coincident with the axis, so the revolve must produce
// a SOLID cylinder (3 faces), not an annulus with a sub-resolution sliver bore.
func TestRevolutionAxisWeldIsScaleRelative(t *testing.T) {
	t.Parallel()
	const rBore, r, h = 1e-4, 1e4, 1e4
	mer := []math.Point2{math.P2(rBore, 0), math.P2(r, 0), math.P2(r, h), math.P2(rBore, h)}
	body, err := brep.SolidOfRevolution(math.P3(0, 0, 0), math.V3(0, 0, 1), mer, "weld")
	if err != nil || body == nil {
		t.Fatalf("SolidOfRevolution(weld) = %v, %v; want a body", body, err)
	}
	got := revVolume(t, body)
	want := stdmath.Pi * r * r * h
	if rel := stdmath.Abs(got-want) / want; rel > 0.03 {
		t.Errorf("welded-bore volume = %.4g, want ≈%.4g (rel %.4f > 3%% band)", got, want, rel)
	}
	if n := len(body.Faces()); n != 3 {
		t.Errorf("welded-bore solid has %d faces, want 3 (wall + 2 disc caps; sub-resolution bore welds to axis)", n)
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
