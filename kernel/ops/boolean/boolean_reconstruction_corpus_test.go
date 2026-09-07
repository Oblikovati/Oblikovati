// SPDX-License-Identifier: GPL-2.0-only

package boolean

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/subd"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The five fixtures the mesh-arrangement RECONSTRUCTION owned, driven through the PUBLIC boolean.
//
// Each was written against reconstructBoolean, the engine that rebuilt an analytic body from the
// faceted boolean's provenance. ADR-0061 stage 7 deletes that engine, so the fixtures come here: the
// same geometry, the same properties, asked of the one general pipeline. They are corpus rows now,
// not engine tests — which is the conversion the ADR promised rather than deleting them to move a
// number.

// TestSteppedShaftKeepsBothAnalyticWalls: two coaxial cylinders of different radius, stacked and
// unioned. Nothing removes either wall, so both survive as analytic cylinders and only the shared cap
// is trimmed to an annulus.
func TestSteppedShaftKeepsBothAnalyticWalls(t *testing.T) {
	t.Parallel()
	lower, err := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3, 5)
	if err != nil {
		t.Fatalf("lower cylinder: %v", err)
	}
	upper, err := brep.SolidCylinder(math.P3(0, 0, 5), math.V3(0, 0, 1), 1.5, 5)
	if err != nil {
		t.Fatalf("upper cylinder: %v", err)
	}
	body := mustBooleanSolid(t, Join, lower, upper)
	if n := cylinderFaceCount(body); n != 2 {
		t.Errorf("stepped shaft has %d analytic cylinder walls, want 2", n)
	}
	assertVolume(t, body, stdmath.Pi*9*5+stdmath.Pi*2.25*5, 1e-6)
}

// TestOverlappingBoxesUnionExactly is the planar counterpart, end to end.
func TestOverlappingBoxesUnionExactly(t *testing.T) {
	t.Parallel()
	a := boxAt(2, 2, 2, math.V3(0, 0, 0), "a")
	b := boxAt(2, 2, 2, math.V3(1, 0, 0), "b") // overlaps a in x: union volume = 12
	assertVolume(t, mustBooleanSolid(t, Join, a, b), 12, 1e-9)
}

// TestCocylindricalCapOnWallStaysAnalytic: a D-profile prism seated on a cylinder of the SAME radius.
// #2167 was that this join FACETED — no analytic cylinder at all — so the two walls' mismatched facet
// grids showed as a visible seam. Both walls are analytic here and lie on ONE surface, so they
// re-tessellate against that surface and the seam is gone.
//
// They are still TWO faces where a correct B-rep has one: their common boundary is part of the
// cylinder's rim, not a whole edge of it, which mergeCoincidentFaces leaves alone (see joinedLoops).
// The count is pinned at 2 rather than relaxed, so landing that merge trips this test and converts it.
func TestCocylindricalCapOnWallStaysAnalytic(t *testing.T) {
	if testing.Short() {
		t.Skip("corpus tier: `make test-corpus`")
	}
	t.Parallel()
	cyl, err := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3, 6)
	if err != nil {
		t.Fatalf("cylinder: %v", err)
	}
	body := mustBooleanSolid(t, Join, cyl, dPrismBody(3, 0.6, 6, 10, "d"))
	if n := cylinderFaceCount(body); n != 2 {
		t.Errorf("cocylindrical join has %d analytic cylinder faces, want 2 — both walls analytic, on "+
			"one surface, pending the partial-boundary merge that makes them one face", n)
	}
	minor := 0.5 * 9 * (1.2 - stdmath.Sin(1.2))
	assertVolume(t, body, stdmath.Pi*9*6+(stdmath.Pi*9-minor)*4, 5e-3)
}

// TestObliqueBoreKeepsItsEllipticalRims: a tilted cylinder bored cleanly through a slab's top and
// bottom faces cuts an ELLIPSE at each. Both rims must stay analytic and the bore wall one cylinder.
func TestObliqueBoreKeepsItsEllipticalRims(t *testing.T) {
	t.Parallel()
	slab, err := brep.SolidBlock(math.P3(-1, -1, 0), math.P3(1, 1, 0.5), "slab")
	if err != nil {
		t.Fatalf("slab: %v", err)
	}
	bore, err := brep.SolidCylinder(math.P3(0, 0, -0.2), math.V3(0.3, 0, 1), 0.2, 1.0)
	if err != nil {
		t.Fatalf("bore: %v", err)
	}
	body := mustBooleanSolid(t, Cut, slab, bore)
	if n := cylinderFaceCount(body); n != 1 {
		t.Errorf("oblique bore kept %d cylinder walls, want 1", n)
	}
	if !hasEllipseEdge(body) {
		t.Error("oblique bore has no analytic elliptical rim edge — it faceted the ellipse")
	}
	// An oblique cylinder through a slab removes π r² times the slant length t/cos θ.
	slant := 0.5 * stdmath.Sqrt(1+0.3*0.3)
	assertVolume(t, body, 2*2*0.5-stdmath.Pi*0.04*slant, 1e-6)
}

// TestObliqueStubOnBoxKeepsItsEllipticalSeam: a tilted cylinder unioned with a box, poking out
// through the box top at an angle. The seam where the wall meets the top face is an ellipse.
func TestObliqueStubOnBoxKeepsItsEllipticalSeam(t *testing.T) {
	t.Parallel()
	box, err := brep.SolidBlock(math.P3(-1, -1, 0), math.P3(1, 1, 1), "box")
	if err != nil {
		t.Fatalf("box: %v", err)
	}
	stub, err := brep.SolidCylinder(math.P3(0, 0, 0.5), math.V3(0.4, 0, 1), 0.3, 1.2)
	if err != nil {
		t.Fatalf("stub: %v", err)
	}
	body := mustBooleanSolid(t, Join, box, stub)
	if n := cylinderFaceCount(body); n < 1 {
		t.Errorf("cyl∪box union kept %d cylinder walls, want at least 1", n)
	}
	if !hasEllipseEdge(body) {
		t.Error("cyl∪box union has no analytic elliptical seam edge — it faceted the ellipse")
	}
	// Requicha's bracket: a union holds at least the larger operand and at most their sum.
	v := query.BodyGeometryProperties(body, PropertyQuality()).Volume
	vBox := query.BodyGeometryProperties(box, PropertyQuality()).Volume
	vStub := query.BodyGeometryProperties(stub, PropertyQuality()).Volume
	if v < vBox || v > vBox+vStub {
		t.Errorf("cyl∪box union volume %.5f is outside [%.5f, %.5f]", v, vBox, vBox+vStub)
	}
}

// TestAnnularRingCutRemovesExactlyAQuarter is the volume half of the ring row (brep carries its
// topology). The ring lies wholly within the box's y span and reaches x, z = ±1.5 inside a 2-unit box,
// so the removed material is exactly a QUARTER of the annulus — an exact number, not a bracket.
func TestAnnularRingCutRemovesExactlyAQuarter(t *testing.T) {
	t.Parallel()
	box, err := brep.SolidBlock(math.P3(0, 0, 0), math.P3(2, 2, 4), "box")
	if err != nil {
		t.Fatalf("SolidBlock: %v", err)
	}
	ring, err := brep.SolidOfRevolution(math.P3(0, 0, 0), math.V3(0, 1, 0),
		[]math.Point2{math.P2(0.5, 0.5), math.P2(1.5, 0.5), math.P2(1.5, 1.5), math.P2(0.5, 1.5)}, "ring")
	if err != nil {
		t.Fatalf("SolidOfRevolution: %v", err)
	}
	body := mustBooleanSolid(t, Cut, box, ring)
	assertVolume(t, body, 2*2*4-stdmath.Pi*(1.5*1.5-0.5*0.5)/4, 1e-9)
}

// mustBooleanSolid runs the public boolean and requires a valid closed manifold solid.
func mustBooleanSolid(t *testing.T, op PartFeatureOperation, target, tool *topo.Body) *topo.Body {
	t.Helper()
	body, err := Boolean(op, target, tool)
	if err != nil {
		t.Fatalf("%s: the general pipeline declined: %v", op, err)
	}
	if body == nil {
		t.Fatalf("%s returned an empty body", op)
	}
	if r := Validate(body); !r.Valid || !r.Closed || !r.Manifold || !body.IsSolid() {
		t.Fatalf("%s is not a valid closed manifold solid: %+v", op, r)
	}
	return body
}

// assertVolume checks the analytic volume against an exact expectation, relative.
func assertVolume(t *testing.T, b *topo.Body, want, relTol float64) {
	t.Helper()
	got := query.BodyGeometryProperties(b, PropertyQuality()).Volume
	if rel := stdmath.Abs(got-want) / want; rel > relTol {
		t.Errorf("volume = %.8f, want %.8f (rel err %.3g > %.3g)", got, want, rel, relTol)
	}
}

// hasEllipseEdge reports whether the body carries an analytic elliptical boundary edge.
func hasEllipseEdge(b *topo.Body) bool {
	for _, e := range b.Edges() {
		switch e.Geometry().(type) {
		case geom.EllipticalArc, geom.EllipseFull:
			return true
		}
	}
	return false
}

// boxAt builds a box translated by off.
func boxAt(sx, sy, sz float64, off math.Vector3, feat string) *topo.Body {
	m := subd.Box(sx, sy, sz)
	for i := range m.Verts {
		m.Verts[i] = m.Verts[i].TranslateBy(off)
	}
	return subd.ToBody(m, feat)
}
