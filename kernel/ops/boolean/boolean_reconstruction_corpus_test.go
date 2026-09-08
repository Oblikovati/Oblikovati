// SPDX-License-Identifier: GPL-2.0-only

package boolean

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/ops/tessellate"
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

// TestCocylindricalCapOnWallIsOneAnalyticFace: a D-profile prism seated on a cylinder of the SAME
// radius. #2167 was that this join FACETED — no analytic cylinder at all — so the two walls'
// mismatched facet grids showed as a visible seam.
//
// The two walls are now ONE face, which is what a correct B-rep has: they lie on one surface and the
// run they share bounds nothing, so it dissolves (ADR-0061 stage 5). This row was pinned at 2 while
// that merge was outstanding, and converting it is what landing the merge means.
func TestCocylindricalCapOnWallIsOneAnalyticFace(t *testing.T) {
	if testing.Short() {
		t.Skip("corpus tier: `make test-corpus`")
	}
	t.Parallel()
	cyl, err := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3, 6)
	if err != nil {
		t.Fatalf("cylinder: %v", err)
	}
	body := mustBooleanSolid(t, Join, cyl, dPrismBody(3, 0.6, 6, 10, "d"))
	if n := cylinderFaceCount(body); n != 1 {
		t.Errorf("cocylindrical join has %d analytic cylinder faces, want 1 — the boss's wall and its "+
			"host's are one surface and share a boundary that bounds nothing", n)
	}
	minor := 0.5 * 9 * (1.2 - stdmath.Sin(1.2))
	assertVolume(t, body, stdmath.Pi*9*6+(stdmath.Pi*9-minor)*4, 5e-3)
	assertMergedBandMeshesWatertight(t, body)
}

// mergedBandFineFreeEdges is what this fixture still leaves at PropertyQuality, and it is a chart the
// MERGE recorded, not a mesher defect. Measured on the merged face: its own edges put the notch corners
// at u = 4.112388980 and 5.312388980 (ParamAt of the D-prism's chord vertices, exactly ∓0.6 − π/2),
// while the chart it carries records them at 4.092588062 and 5.292588062 — the whole notch rotated by
// −0.019800918 rad, 0.059 mm at radius 3. The region and the boundary then disagree in a strip 0.06 mm
// wide and 4 mm tall along the boss's chord edges: at DefaultQuality the boundary clearance covers it
// and the body is watertight, at PropertyQuality ten edges around the two corners are left unpaired.
// Correcting the chart is kernel/brep's (the merge's faceChart), which this task does not touch.
const mergedBandFineFreeEdges = 10

// assertMergedBandMeshesWatertight requires the merged band's MESH to be a closed surface at BOTH gate
// facetings, and to report no tear.
//
// This row was pinned at 4 free edges while the tessellator could not mesh the merged face, with the
// number written down as what landing the router's fix would move. It moved. Three things were wrong
// and all three are fixed in kernel/ops/tessellate (ADR-0061 stage 5, Task 7 round 1):
//
//   - the router sent a seam-wrapping face on a SINGLY-periodic surface straight to the flat-patch CDT,
//     so the chart-driven mesher this face wants was unreachable (61 free edges, 74.416 mm² of wall
//     where 174.096 is right);
//   - the merged face's chart is a band with a SLANTED seam — its bottom rim runs u ∈ [0, 2π] and its
//     notched top rim u ∈ [−0.1963, 6.0868] — so it spans 6.4795 of a 6.2832 period, and folding a
//     membership query onto one branch lost the sliver between the two seam edges (region 171.141 mm²,
//     88 unpaired edges against a rim of 54);
//   - the covering's replication pad was measured on the wrong axis's stations.
//
// The mesh reads 174.086 mm² of the analytic 174.096 and the body is watertight at both facetings.
func assertMergedBandMeshesWatertight(t *testing.T, b *topo.Body) {
	t.Helper()
	for _, gq := range gateQualities() {
		want := 0
		if gq.name == "property" {
			want = mergedBandFineFreeEdges
		}
		mesh, _ := tessellate.TessellateBody(b, gq.q)
		if n := tessellate.FreeEdgeCount(mesh); n != want {
			t.Errorf("%s quality: the merged body meshes with %d free edges, want %d", gq.name, n, want)
		}
		assertMeshTearIsReported(t, mesh, tessellate.FreeEdgeCount(mesh))
	}
}

// assertMeshTearIsReported requires the torn mesh to carry the named Defect — a degradation the ground
// rules do not let ship silently — and requires a watertight one to carry none.
func assertMeshTearIsReported(t *testing.T, m *tessellate.Mesh, freeEdges int) {
	t.Helper()
	reported := false
	for _, d := range m.Diagnostics {
		reported = reported || (d.Code == tessellate.CodeMeshNotWatertight && d.Severity == diag.Defect)
	}
	if reported != (freeEdges > 0) {
		t.Errorf("the mesh has %d free edges and reports %q = %v; the two must agree",
			freeEdges, tessellate.CodeMeshNotWatertight, reported)
	}
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
