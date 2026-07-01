// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Steinmetz through the GENERAL (u,v)-arrangement pipeline (#1403, approach A). The bicylinder is built by
// splitting the self-intersecting imprint into four open elliptical arcs and trimming each cylinder side —
// the angular-next-edge tracer separating the lobes — instead of the bespoke loop→body assembler. The
// result must match the bespoke topology: a watertight four-face solid of cylinder faces and elliptical-arc
// edges meeting at the two pinch vertices.

// TestSteinmetzIntersectGeneralWatertight pins that the general intersect produces the same watertight
// four-face bicylinder as the bespoke constructor: cylinder faces, every edge used exactly twice.
func TestSteinmetzIntersectGeneralWatertight(t *testing.T) {
	cx, _ := SolidCylinder(math.P3(-6, 0, 0), math.V3(1, 0, 0), 3, 12)
	cz, _ := SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 3, 12)

	res, ok := SteinmetzIntersectGeneral(cx, cz, nil)
	if !ok {
		t.Fatal("general Steinmetz intersect declined; want a four-face bicylinder")
	}
	if !res.IsSolid() {
		t.Fatalf("general Steinmetz result is not a solid: %+v", res)
	}
	for _, f := range res.Faces() {
		if _, isCyl := f.Geometry().(geom.Cylinder); !isCyl {
			t.Errorf("face surface %T is not a cylinder (the analytic surface must survive)", f.Geometry())
		}
	}
	for _, e := range res.Edges() {
		if uses := len(e.Uses()); uses != 2 {
			t.Errorf("edge %v has %d uses, want 2 (a closed manifold — no free/non-manifold edge)", e.Lineage(), uses)
		}
	}
	if n := len(res.Faces()); n != 4 {
		t.Errorf("general Steinmetz has %d faces, want 4 (two lobes per cylinder)", n)
	}
}

// TestSteinmetzGeneralEdgesAnchoredToVertices pins the well-formedness invariant a boolean must never
// violate: every stored edge curve starts at its StartVertex and ends at its EndVertex (PointAt(0)≈start,
// PointAt(1)≈end for a forward use). A lobe loop walks its shared elliptical arc in the arc's DECREASING
// parameter direction (t0=1→t1=0); if edgeCurveFor keeps the arc's original forward parameterisation, the
// stored curve's PointAt(0) lands at the FAR pinch — 2R away from StartVertex — so the face's discretised
// boundary jumps across the solid, the (u,v) loop self-intersects, and the tessellator falls back to a
// mis-oriented plane patch (volume 37 vs 144). Re-anchoring the sub-arc to [t0,t1] restores the invariant.
func TestSteinmetzGeneralEdgesAnchoredToVertices(t *testing.T) {
	cx, _ := SolidCylinder(math.P3(-6, 0, 0), math.V3(1, 0, 0), 3, 12)
	cz, _ := SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 3, 12)

	res, ok := SteinmetzIntersectGeneral(cx, cz, nil)
	if !ok {
		t.Fatal("general Steinmetz intersect declined")
	}
	const tol = 1e-6
	for _, e := range res.Edges() {
		c := e.Geometry()
		if d := float64(c.PointAt(0).DistanceTo(e.StartVertex().Point())); d > tol {
			t.Errorf("edge curve PointAt(0) is %g from StartVertex (arc parameterised opposite to its vertices)", d)
		}
		if d := float64(c.PointAt(1).DistanceTo(e.EndVertex().Point())); d > tol {
			t.Errorf("edge curve PointAt(1) is %g from EndVertex", d)
		}
	}
}

// TestSteinmetzGeneralDeclinesUnequalRadius pins that the general path declines the non-Steinmetz case (so
// kernel/ops keeps the clean crossing-cylinder pipeline for unequal radii).
func TestSteinmetzGeneralDeclinesUnequalRadius(t *testing.T) {
	cx, _ := SolidCylinder(math.P3(-6, 0, 0), math.V3(1, 0, 0), 1.5, 12)
	cz, _ := SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 3, 12)
	if _, ok := SteinmetzIntersectGeneral(cx, cz, nil); ok {
		t.Error("general Steinmetz must decline unequal radii (ok=false)")
	}
}
