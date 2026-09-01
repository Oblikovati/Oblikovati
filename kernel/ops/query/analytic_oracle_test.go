// SPDX-License-Identifier: GPL-2.0-only

package query

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The kernel's mass-properties and box oracles read the analytic B-rep (M48/C3 #3421/#3453/#3454/
// #3482). These tests assert what a tessellated integral CANNOT deliver: exact closed-form values.

// analyticQuadRelTol is what "exact" means for these assertions: the adaptive quadrature converges
// to its own relative tolerance (quadRelTol), so the analytic values agree with the closed forms to
// ~1e-11 relative. The tessellated sums they replaced were wrong in the THIRD digit.
const analyticQuadRelTol = 1e-9

// boundaryIndex is one operand prepared for repeated "does this point lie on the boundary?"
// questions: its faces, with a box tree over their range boxes, so a probe only reaches the faces
// whose box covers it. It is built ONCE per boolean — scanning every face per probe made the gate
// cost O(result faces × operand faces) exact projections, which on a body of tens of thousands of
// faces costs far more than the boolean it is checking.
type boundaryIndex struct {
	body    *topo.Body
	faces   []*topo.Face
	tree    *geom.BoxTree
	unboxed []*topo.Face // faces with no range box: a BOUNDARY-LESS face has no vertices to build one
}

// analyticRelDiff is |got-want|/|want|, the scale-free comparison these closed-form checks need.
func analyticRelDiff(got, want float64) float64 {
	if want == 0 {
		return stdmath.Abs(got)
	}
	return stdmath.Abs(got-want) / stdmath.Abs(want)
}

// TestBodyGeometryPropertiesIsExactForACylinder: πr²h and 2πr(r+h) to the last digits, where the
// tessellated sum was low by the chord deficit of its inscribed polygon.
func TestBodyGeometryPropertiesIsExactForACylinder(t *testing.T) {
	t.Parallel()
	const r, h = 3.0, 7.0
	body, err := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), r, h)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	gp := BodyGeometryProperties(body, DefaultQuality())
	if want := stdmath.Pi * r * r * h; analyticRelDiff(gp.Volume, want) > analyticQuadRelTol {
		t.Errorf("volume = %.12f, want πr²h = %.12f", gp.Volume, want)
	}
	if want := 2 * stdmath.Pi * r * (r + h); analyticRelDiff(gp.Area, want) > analyticQuadRelTol {
		t.Errorf("area = %.12f, want 2πr(r+h) = %.12f", gp.Area, want)
	}
	if d := float64(gp.Centroid.DistanceTo(math.P3(0, 0, h/2))); d > 1e-9 {
		t.Errorf("centroid is %g off the axis midpoint", d)
	}
}

// TestBodyInertiaIsExactForACylinder: Izz = ½Vr² about the axis, from the analytic integral.
func TestBodyInertiaIsExactForACylinder(t *testing.T) {
	t.Parallel()
	const r, h = 3.0, 7.0
	body, err := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), r, h)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	got := BodyInertia(body, DefaultQuality()).Izz
	want := 0.5 * (stdmath.Pi * r * r * h) * r * r
	if analyticRelDiff(got, want) > analyticQuadRelTol {
		t.Errorf("Izz = %.12f, want ½Vr² = %.12f", got, want)
	}
}

// TestPreciseRangeBoxSpansTheFullSphere: the equator bulges past every boundary curve, so a box
// read off facet chords is short by the sagitta on all six faces. The analytic box is not.
func TestPreciseRangeBoxSpansTheFullSphere(t *testing.T) {
	t.Parallel()
	const r = 5.0
	body, err := brep.SolidSphere(math.P3(1, 2, 3), r, "ball")
	if err != nil {
		t.Fatalf("SolidSphere: %v", err)
	}
	box := PreciseRangeBox(body, DefaultQuality())
	if stdmath.Abs(float64(box.Min.X)-(1-r)) > 1e-9 || stdmath.Abs(float64(box.Max.Z)-(3+r)) > 1e-9 {
		t.Errorf("sphere box = %v, want the centre ±%g on every axis", box, r)
	}
}

// TestPreciseRangeBoxSpansACylinderRadius: the side face's rim circles carry the bulge, and their
// closed-form extrema put the box on the true radius rather than on an inscribed polygon's apothem.
func TestPreciseRangeBoxSpansACylinderRadius(t *testing.T) {
	t.Parallel()
	const r, h = 3.0, 7.0
	body, err := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), r, h)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	box := PreciseRangeBox(body, DefaultQuality())
	if stdmath.Abs(float64(box.Max.X)-r) > 1e-9 || stdmath.Abs(float64(box.Min.Y)+r) > 1e-9 {
		t.Errorf("cylinder box = %v, want ±%g in x and y", box, r)
	}
	if stdmath.Abs(float64(box.Min.Z)) > 1e-9 || stdmath.Abs(float64(box.Max.Z)-h) > 1e-9 {
		t.Errorf("cylinder box z = [%g, %g], want [0, %g]", box.Min.Z, box.Max.Z, h)
	}
}

// TestVectorAreaClosureRejectsAResidual: the outward vector area of a closed surface is exactly
// zero, so a residual means some face was integrated over the wrong region or with a flipped
// orientation. The check must reject that rather than let a wrong number through.
func TestVectorAreaClosureRejectsAResidual(t *testing.T) {
	t.Parallel()
	exact, err := brep.SolidBlock(math.P3(0, 0, 0), math.P3(1, 1, 1), "exact")
	if err != nil {
		t.Fatalf("SolidBlock: %v", err)
	}
	if !vectorAreaCloses(exact, MassTerms{Area: 100}) {
		t.Error("a body whose vector area vanishes must pass the closure check")
	}
	if vectorAreaCloses(exact, MassTerms{Area: 100, Ax: 1}) {
		t.Error("a body with a 1%% vector-area residual must be declined, not integrated")
	}
	if vectorAreaCloses(exact, MassTerms{}) {
		t.Error("a body with no area at all cannot be certified closed")
	}
	if s := AchievedBoundarySlack(exact); s != 0 {
		t.Errorf("an all-analytic body claimed %g of boundary slack; its edges are exact", s)
	}
}

// TestGreenFormFollowsTheClosingAxis: the reduction is chosen by which parameter the loops return
// to, and a loop that closes in neither is refused rather than integrated over a region that is not
// bounded in the covering space.
func TestGreenFormFollowsTheClosingAxis(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		loop     faceLoop
		wantDV   bool
		wantOKay bool
	}{
		{"closes both ways", faceLoop{}, true, true},
		{"wraps u, closes v", faceLoop{netU: 4 * stdmath.Pi}, false, true},
		{"wraps v, closes u", faceLoop{netV: 2 * stdmath.Pi}, true, true},
		{"wraps both", faceLoop{netU: 2 * stdmath.Pi, netV: 2 * stdmath.Pi}, false, false},
	}
	for _, c := range cases {
		form, ok := greenFormFor([]faceLoop{c.loop})
		if ok != c.wantOKay || (ok && form.dv != c.wantDV) {
			t.Errorf("%s: form.dv=%v ok=%v, want dv=%v ok=%v", c.name, form.dv, ok, c.wantDV, c.wantOKay)
		}
	}
}

// TestBoundaryIndexFindsABoundarylessFace: the box tree cannot return a face with no range box, and
// a BOUNDARY-LESS face — the whole sphere a ball is made of — has none, because a range box is built
// from vertices and edge curves. Those faces must still answer a boundary probe, or every coaxial
// sphere boolean reads as fabricated.
func TestBoundaryIndexFindsABoundarylessFace(t *testing.T) {
	t.Parallel()
	ball, err := brep.SolidSphere(math.P3(0, 0, 0), 5, "ball")
	if err != nil {
		t.Fatalf("SolidSphere: %v", err)
	}
	bi := newBoundaryIndex(ball)
	if len(bi.unboxed) != 1 {
		t.Fatalf("ball indexed %d unboxed faces, want its single boundary-less sphere", len(bi.unboxed))
	}
	tol := geom.ResolutionForBox(ball.RangeBox()).Sew()
	if !bi.on(math.P3(0, 5, 0), tol) {
		t.Error("a point on the ball's own surface did not register as on its boundary")
	}
	if bi.on(math.P3(0, 4, 0), tol) {
		t.Error("a point strictly inside the ball registered as on its boundary")
	}
}

// TestBoundaryIndexUsesTheTreeForBoundedFaces: a bounded face is found through the box tree, and a
// point well away from every face is not.
func TestBoundaryIndexUsesTheTreeForBoundedFaces(t *testing.T) {
	t.Parallel()
	block, err := brep.SolidBlock(math.P3(0, 0, 0), math.P3(2, 2, 2), "block")
	if err != nil {
		t.Fatalf("SolidBlock: %v", err)
	}
	bi := newBoundaryIndex(block)
	if len(bi.unboxed) != 0 || len(bi.faces) != 6 {
		t.Fatalf("block indexed %d bounded / %d unboxed faces, want 6 / 0", len(bi.faces), len(bi.unboxed))
	}
	tol := geom.ResolutionForBox(block.RangeBox()).Sew()
	if !bi.on(math.P3(1, 1, 2), tol) {
		t.Error("a point on the block's top face did not register as on its boundary")
	}
	if bi.on(math.P3(1, 1, 1), tol) {
		t.Error("the block's centre registered as on its boundary")
	}
}

// newBoundaryIndex prepares b's faces for boundary probes. A face with an empty range box cannot be
// found by a tree query, so those are kept aside and always tested — that is the whole sphere a ball
// is made of, exactly the face a coaxial sphere/rod boolean has to certify.
func newBoundaryIndex(b *topo.Body) *boundaryIndex {
	bi := &boundaryIndex{body: b}
	var boxes []math.Box
	for _, f := range b.Faces() {
		if box := f.RangeBox(); !box.IsEmpty() {
			bi.faces = append(bi.faces, f)
			boxes = append(boxes, box)
			continue
		}
		bi.unboxed = append(bi.unboxed, f)
	}
	bi.tree = geom.NewBoxTree(boxes)
	return bi
}

// on reports whether p lies on any of the body's trimmed faces, within tol.
func (bi *boundaryIndex) on(p math.Point3, tol float64) bool {
	for _, f := range bi.unboxed {
		if brep.PointOnFace(f, p, tol) {
			return true
		}
	}
	found := false
	bi.tree.Query(pointReach(p, tol), func(i int) bool {
		found = brep.PointOnFace(bi.faces[i], p, tol)
		return found
	})
	return found
}

// pointReach is the query box for a probe: the point grown by the on-boundary tolerance.
func pointReach(p math.Point3, tol float64) math.Box {
	t := math.Scalar(tol)
	return math.NewBox(math.P3(p.X-t, p.Y-t, p.Z-t), math.P3(p.X+t, p.Y+t, p.Z+t))
}
