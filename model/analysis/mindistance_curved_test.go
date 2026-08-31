// SPDX-License-Identifier: GPL-2.0-only

package analysis

import (
	"math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	gmath "oblikovati.org/math"
)

// Curved-face minimum distance (M48/C3 #3458): the closest approach on a convex curved face is a
// single generatrix/point that generically falls BETWEEN tessellation vertices, so a triangle-mesh
// measure is biased by the chord sagitta (~R·(1−cos(π/n)) ≈ 5e-3 cm at default quality here). Every
// case below asserts the exact analytic clearance to 1e-6 mm — a bound the retired tessellated path
// could not meet — proving the measure now reads the exact trimmed B-rep.

// TestMinDistanceTwoSpheres measures two disjoint spheres: the surface clearance is the centre gap
// minus both radii. A whole sphere is one boundaryless face, so this drives the interior
// surface–surface solver (alternating projection) with no boundary edges to lean on.
func TestMinDistanceTwoSpheres(t *testing.T) {
	a, err := brep.SolidSphere(gmath.P3(0, 0, 0), 1, "a")
	if err != nil {
		t.Fatalf("SolidSphere a: %v", err)
	}
	b, err := brep.SolidSphere(gmath.P3(5, 0, 0), 1.5, "b")
	if err != nil {
		t.Fatalf("SolidSphere b: %v", err)
	}
	got := MinDistanceMm(MeasureEntity{Face: a.Faces()[0]}, MeasureEntity{Face: b.Faces()[0]}, ops.DefaultQuality())
	if want := (5.0 - 1.0 - 1.5) * cmToMM; math.Abs(got-want) > 1e-6 {
		t.Errorf("sphere–sphere clearance = %.9f mm, want %.9f (exact, sagitta-free)", got, want)
	}
}

// TestMinDistanceParallelCylinderFaces measures the lateral faces of two parallel cylinders offset
// along +y — the closest generatrices sit at angle 90° from the angle-0 seam, i.e. squarely between
// tessellation vertices, where a faceted side would over-report the gap.
func TestMinDistanceParallelCylinderFaces(t *testing.T) {
	c1, err := brep.SolidCylinder(gmath.P3(0, 0, 0), gmath.V3(0, 0, 1), 1, 4)
	if err != nil {
		t.Fatalf("SolidCylinder c1: %v", err)
	}
	c2, err := brep.SolidCylinder(gmath.P3(0, 5, 0), gmath.V3(0, 0, 1), 1.5, 4)
	if err != nil {
		t.Fatalf("SolidCylinder c2: %v", err)
	}
	got := MinDistanceMm(
		MeasureEntity{Face: lateralFace(t, c1)}, MeasureEntity{Face: lateralFace(t, c2)}, ops.DefaultQuality())
	if want := (5.0 - 1.0 - 1.5) * cmToMM; math.Abs(got-want) > 1e-6 {
		t.Errorf("cylinder-side clearance = %.9f mm, want %.9f (exact, sagitta-free)", got, want)
	}
}

// TestMinDistanceTwoCircleEdges measures two coplanar circle edges (the cylinder top rims): the exact
// circle–circle gap, driving the curve→curve path with a genuinely curved trimmed curve.
func TestMinDistanceTwoCircleEdges(t *testing.T) {
	c1, err := brep.SolidCylinder(gmath.P3(0, 0, 0), gmath.V3(0, 0, 1), 1, 4)
	if err != nil {
		t.Fatalf("SolidCylinder c1: %v", err)
	}
	c2, err := brep.SolidCylinder(gmath.P3(0, 5, 0), gmath.V3(0, 0, 1), 1.5, 4)
	if err != nil {
		t.Fatalf("SolidCylinder c2: %v", err)
	}
	got := MinDistanceMm(
		MeasureEntity{Edge: circleEdge(t, c1)}, MeasureEntity{Edge: circleEdge(t, c2)}, ops.DefaultQuality())
	if want := (5.0 - 1.0 - 1.5) * cmToMM; math.Abs(got-want) > 1e-6 {
		t.Errorf("circle–circle gap = %.9f mm, want %.9f (exact, sagitta-free)", got, want)
	}
}

// lateralFace returns the cylinder's single curved (cylindrical) face.
func lateralFace(t *testing.T, body *topo.Body) *topo.Face {
	t.Helper()
	for _, f := range body.Faces() {
		if _, ok := f.Geometry().(geom.Cylinder); ok {
			return f
		}
	}
	t.Fatalf("no cylindrical face on body")
	return nil
}

// circleEdge returns one circular rim edge of the cylinder.
func circleEdge(t *testing.T, body *topo.Body) *topo.Edge {
	t.Helper()
	for _, e := range body.Edges() {
		if _, ok := e.Geometry().(geom.Circle); ok {
			return e
		}
	}
	t.Fatalf("no circular edge on body")
	return nil
}
