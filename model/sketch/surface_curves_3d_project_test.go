// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	stdmath "math"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/geom"
	gmath "oblikovati.org/math"
)

// TestProjectToSurface3DAlongVectorPiercesAtDirection: a source line at z=5 projected onto the z=0
// plane along (1,0,-1) lands where each ray pierces the plane — shifted +x by the drop height, NOT
// the perpendicular foot directly below (#1841).
func TestProjectToSurface3DAlongVectorPiercesAtDirection(t *testing.T) {
	s := NewSketches3D().Add()
	src := geom.NewLineSegment(gmath.P3(0, 0, 5), gmath.P3(2, 0, 5))
	plane, err := geom.NewPlane(gmath.P3(0, 0, 0), gmath.V3(0, 0, 1))
	if err != nil {
		t.Fatalf("NewPlane: %v", err)
	}
	c := s.AddProjectToSurfaceCurve3DAlongVector(src, StaticSurface(plane), gmath.V3(1, 0, -1))
	if c.Projection != types.ProjectAlongVector {
		t.Errorf("Projection = %v, want alongVector", c.Projection)
	}

	pts := c.Evaluate()
	if len(pts) == 0 {
		t.Fatal("alongVector projection produced no points")
	}
	// Sample at the source start (0,0,5): the ray (1,0,-1) pierces z=0 at (5,0,0).
	if !pts[0].IsEqualTo(gmath.P3(5, 0, 0), 1e-6) {
		t.Errorf("first pierce = %v, want (5,0,0) — along the ray, not the foot (0,0,0)", pts[0])
	}
	for _, p := range pts {
		if stdmath.Abs(float64(p.Z)) > 1e-6 {
			t.Errorf("pierce %v not on the z=0 plane", p)
		}
	}
}

// TestProjectToSurface3DWrapUnwrapsOntoCylinder: the wrap projection maps a planar source line onto
// a cylinder preserving arc length — planar x becomes angle u = x/R (#1841). Exercised through the
// model entity so the sketch wiring (Frame field + wrap branch of Evaluate) is covered.
func TestProjectToSurface3DWrapUnwrapsOntoCylinder(t *testing.T) {
	const r = 2.0
	s := NewSketches3D().Add()
	cyl, err := geom.NewCylinderWithRef(gmath.P3(0, 0, 0), gmath.V3(0, 0, 1), gmath.V3(1, 0, 0), r)
	if err != nil {
		t.Fatalf("NewCylinderWithRef: %v", err)
	}
	arc := stdmath.Pi * r / 2
	src := geom.NewLineSegment(gmath.P3(r, 0, 0), gmath.P3(r, arc, 0))
	frame := geom.WrapFrame{Origin: gmath.P3(r, 0, 0), U: gmath.V3(0, 1, 0), V: gmath.V3(0, 0, 1)}

	c := s.AddProjectToSurfaceCurve3DWrap(src, StaticSurface(cyl), frame)
	if c.Projection != types.ProjectWrapToSurface {
		t.Errorf("Projection = %v, want wrap", c.Projection)
	}
	pts := c.Evaluate()
	if len(pts) == 0 {
		t.Fatal("wrap projection produced no points")
	}
	if !pts[0].IsEqualTo(gmath.P3(r, 0, 0), 1e-6) {
		t.Errorf("wrap start = %v, want the anchor (2,0,0)", pts[0])
	}
	// A quarter-turn's worth of planar length lands at (0, R, 0), a quarter around the cylinder.
	if !pts[len(pts)-1].IsEqualTo(gmath.P3(0, r, 0), 1e-6) {
		t.Errorf("wrap end = %v, want (0,2,0)", pts[len(pts)-1])
	}
}

// TestProjectToSurface3DClosestPointUnchanged: the default projection still drops each sample to its
// perpendicular foot (#1841 — no behaviour change for existing callers).
func TestProjectToSurface3DClosestPointUnchanged(t *testing.T) {
	s := NewSketches3D().Add()
	src := geom.NewLineSegment(gmath.P3(1, 2, 5), gmath.P3(3, 2, 5))
	plane, _ := geom.NewPlane(gmath.P3(0, 0, 0), gmath.V3(0, 0, 1))
	c := s.AddProjectToSurfaceCurve3DRef(src, StaticSurface(plane))

	pts := c.Evaluate()
	// The foot of (1,2,5) on z=0 is (1,2,0) — directly below.
	if !pts[0].IsEqualTo(gmath.P3(1, 2, 0), 1e-6) {
		t.Errorf("closest-point foot = %v, want (1,2,0)", pts[0])
	}
}
