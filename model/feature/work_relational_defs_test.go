// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"oblikovati.org/math"
)

// Relational datum-axis (#1840) and datum-point (#1842) constructors: analytic checks that each
// builds the expected geometry and reports degenerate input as unhealthy.

func TestAxisPointAndPlane(t *testing.T) {
	g := NewWorkGeometry()
	pt := g.WorkPoints().AddByPosition(func() math.Point3 { return math.P3(0, 0, 5) })
	wa := g.WorkAxes().AddByPointAndPlane(pt.Key(), OriginXYPlane)
	if !wa.Health().OK() {
		t.Fatalf("point-and-plane axis sick: %+v", wa.Health())
	}
	if !wa.Origin().IsEqualTo(math.P3(0, 0, 5), wtol) {
		t.Errorf("origin = %v, want (0,0,5)", wa.Origin())
	}
	if !wa.Direction().AsVector().IsParallelTo(math.V3(0, 0, 1), wtol) {
		t.Errorf("direction = %v, want +Z (the XY normal)", wa.Direction())
	}
}

func TestAxisLineAndPoint(t *testing.T) {
	g := NewWorkGeometry()
	pt := g.WorkPoints().AddByPosition(func() math.Point3 { return math.P3(0, 5, 0) })
	wa := g.WorkAxes().AddByLineAndPoint(OriginXAxis, pt.Key())
	if !wa.Health().OK() {
		t.Fatalf("line-and-point axis sick: %+v", wa.Health())
	}
	if !wa.Origin().IsEqualTo(math.P3(0, 5, 0), wtol) || !wa.Direction().AsVector().IsParallelTo(math.V3(1, 0, 0), wtol) {
		t.Errorf("axis = %v dir %v, want through (0,5,0) parallel to X", wa.Origin(), wa.Direction())
	}
}

func TestAxisLineAndPlane(t *testing.T) {
	g := NewWorkGeometry()
	// A grounded line at 45° in the XZ plane, projected onto XY, becomes the X axis.
	diag, _ := math.NewUnitVector3(1, 0, 1)
	line := g.WorkAxes().AddByLine(math.P3(0, 0, 2), diag)
	wa := g.WorkAxes().AddByLineAndPlane(line.Key(), OriginXYPlane)
	if !wa.Health().OK() {
		t.Fatalf("line-and-plane axis sick: %+v", wa.Health())
	}
	if !wa.Direction().AsVector().IsParallelTo(math.V3(1, 0, 0), wtol) {
		t.Errorf("projected direction = %v, want +X", wa.Direction())
	}
	if o := wa.Origin(); o.Z > wtol || o.Z < -wtol {
		t.Errorf("projected origin Z = %v, want on the XY plane (0)", o.Z)
	}
}

func TestAxisLineAndPlanePerpendicularIsDegenerate(t *testing.T) {
	g := NewWorkGeometry()
	line := g.WorkAxes().AddByLine(math.P3(0, 0, 3), mustUnit(0, 0, 1)) // ⟂ to XY
	wa := g.WorkAxes().AddByLineAndPlane(line.Key(), OriginXYPlane)
	if wa.Health().OK() {
		t.Error("a line perpendicular to the plane should be degenerate")
	}
}

func TestPointByPoint(t *testing.T) {
	g := NewWorkGeometry()
	src := g.WorkPoints().AddByPosition(func() math.Point3 { return math.P3(1, 2, 3) })
	wp := g.WorkPoints().AddByPoint(src.Key())
	if !wp.Point().IsEqualTo(math.P3(1, 2, 3), wtol) {
		t.Errorf("point = %v, want (1,2,3)", wp.Point())
	}
}

func TestPointTwoLinesIntersectAtOrigin(t *testing.T) {
	g := NewWorkGeometry()
	wp := g.WorkPoints().AddByTwoLines(OriginXAxis, OriginYAxis)
	if !wp.Health().OK() {
		t.Fatalf("two-lines point sick: %+v", wp.Health())
	}
	if !wp.Point().IsEqualTo(math.P3(0, 0, 0), wtol) {
		t.Errorf("X∩Y = %v, want the origin", wp.Point())
	}
}

func TestPointTwoLinesSkewIsDegenerate(t *testing.T) {
	g := NewWorkGeometry()
	// The X axis and a Y-parallel line lifted to z=5 never meet.
	skew := g.WorkAxes().AddByLine(math.P3(0, 0, 5), mustUnit(0, 1, 0))
	wp := g.WorkPoints().AddByTwoLines(OriginXAxis, skew.Key())
	if wp.Health().OK() {
		t.Error("skew lines should be degenerate")
	}
}

func TestPointThreePlanesAtOrigin(t *testing.T) {
	g := NewWorkGeometry()
	wp := g.WorkPoints().AddByThreePlanes(OriginXYPlane, OriginXZPlane, OriginYZPlane)
	if !wp.Health().OK() {
		t.Fatalf("three-planes point sick: %+v", wp.Health())
	}
	if !wp.Point().IsEqualTo(math.P3(0, 0, 0), wtol) {
		t.Errorf("XY∩XZ∩YZ = %v, want the origin", wp.Point())
	}
}

func TestPointThreePlanesParallelIsDegenerate(t *testing.T) {
	g := NewWorkGeometry()
	off := g.WorkPlanes().AddByPlaneAndOffset(OriginXYPlane, func() float64 { return 5 }) // parallel to XY
	wp := g.WorkPoints().AddByThreePlanes(OriginXYPlane, off.Key(), OriginXZPlane)
	if wp.Health().OK() {
		t.Error("two parallel planes should make the three-plane point degenerate")
	}
}
