// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Equal-radius Steinmetz intersection (M2 Phase 2, Oblikovati/Oblikovati#1335). Two equal-radius
// perpendicular cylinders intersect in the bicylinder: four cylindrical lobe faces sharing four elliptical
// arcs and two pinch vertices. Volume is checked through ops_test; here the concern is the watertight
// topology and that the surfaces and edges stay analytic (cylinder faces, elliptical-arc edges).

// TestSteinmetzIsWatertightFourFaces intersects two radius-3 cylinders (axes x and z) and checks the result
// is a watertight four-face solid: every face a cylinder, every edge an elliptical arc used exactly twice,
// two pinch vertices.
func TestSteinmetzIsWatertightFourFaces(t *testing.T) {
	cx, _ := SolidCylinder(math.P3(-6, 0, 0), math.V3(1, 0, 0), 3, 12)
	cz, _ := SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 3, 12)

	res, ok := EqualRadiusSteinmetzIntersect(cx, cz)
	if !ok {
		t.Fatal("Steinmetz intersection declined; want a four-face bicylinder")
	}
	if !res.IsSolid() {
		t.Fatalf("Steinmetz result is not a solid: %+v", res)
	}
	if n := len(res.Faces()); n != 4 {
		t.Errorf("Steinmetz has %d faces, want 4 (two lobes per cylinder)", n)
	}
	for _, f := range res.Faces() {
		if _, isCyl := f.Geometry().(geom.Cylinder); !isCyl {
			t.Errorf("face surface %T is not a cylinder", f.Geometry())
		}
	}
	for _, e := range res.Edges() {
		if uses := len(e.Uses()); uses != 2 {
			t.Errorf("edge %v has %d uses, want 2 (a closed manifold)", e.Lineage(), uses)
		}
		if _, isArc := e.Geometry().(geom.EllipticalArc); !isArc {
			t.Errorf("edge %v geometry %T is not an elliptical arc", e.Lineage(), e.Geometry())
		}
	}
	if n := len(res.Edges()); n != 4 {
		t.Errorf("Steinmetz has %d edges, want 4 (two ellipses, each split at the pinch points)", n)
	}
	if n := len(res.Vertices()); n != 2 {
		t.Errorf("Steinmetz has %d vertices, want 2 (the pinch points)", n)
	}
}

// TestSteinmetzUnequalRadiusDefers: unequal radii are the clean thin-through-fat case (handled by the SSI
// imprint path), not Steinmetz, so this assembler declines.
func TestSteinmetzUnequalRadiusDefers(t *testing.T) {
	cx, _ := SolidCylinder(math.P3(-6, 0, 0), math.V3(1, 0, 0), 1.5, 12)
	cz, _ := SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 3, 12)
	if _, ok := EqualRadiusSteinmetzIntersect(cx, cz); ok {
		t.Error("unequal-radius crossing should defer from the Steinmetz assembler (ok=false)")
	}
}

// TestSteinmetzNonCrossingDefers: equal-radius cylinders whose axes do not intersect (skew/offset) are not
// the Steinmetz case.
func TestSteinmetzNonCrossingDefers(t *testing.T) {
	cx, _ := SolidCylinder(math.P3(-6, 0, 5), math.V3(1, 0, 0), 3, 12) // offset in z, axes do not meet
	cz, _ := SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 3, 12)
	if _, ok := EqualRadiusSteinmetzIntersect(cx, cz); ok {
		t.Error("non-crossing equal-radius cylinders should defer (ok=false)")
	}
}
