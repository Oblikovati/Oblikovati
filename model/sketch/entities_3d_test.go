// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"math"
	"testing"

	gmath "github.com/Oblikovati/oblikovati/math"
)

func zAxis() gmath.UnitVector3 {
	u, _ := gmath.NewUnitVector3(0, 0, 1)
	return u
}

// TestLine3D checks a 3D line registers two endpoints as solver variables and reports
// its geometry, and that its kernel segment spans the endpoints.
func TestLine3D(t *testing.T) {
	s := NewSketches3D().Add()
	l := s.AddLine3D(gmath.P3(0, 0, 0), gmath.P3(3, 0, 4))

	if s.EntityCount() != 1 || len(s.AllPoints3D()) != 2 {
		t.Fatalf("line should add 1 entity and 2 points, got %d/%d", s.EntityCount(), len(s.AllPoints3D()))
	}
	if float64(l.Length()) != 5 {
		t.Errorf("Length = %v, want 5 (3-4-5)", l.Length())
	}
	if d := l.Direction(); float64(d.X) != 3 || float64(d.Z) != 4 {
		t.Errorf("Direction = %v, want (3,0,4)", d)
	}
	if s.DegreesOfFreedom() != 6 {
		t.Errorf("a free 3D line has 6 DOF, got %d", s.DegreesOfFreedom())
	}
	seg := l.Segment()
	if seg.StartPoint != l.A.Position() || seg.EndPoint != l.B.Position() {
		t.Errorf("Segment endpoints %v..%v, want %v..%v", seg.StartPoint, seg.EndPoint, l.A.Position(), l.B.Position())
	}
}

// TestCircle3D checks a 3D circle's radius is an extra solver DOF and its kernel curve
// has the requested radius and axis.
func TestCircle3D(t *testing.T) {
	s := NewSketches3D().Add()
	c := s.AddCircle3D(gmath.P3(1, 2, 3), zAxis(), 5)

	if float64(c.CircleRadius()) != 5 || c.CenterPoint() != c.Center {
		t.Fatalf("circle accessors wrong: r=%v", c.CircleRadius())
	}
	// center (3) + radius (1) = 4 DOF.
	if s.DegreesOfFreedom() != 4 {
		t.Errorf("a free 3D circle has 4 DOF, got %d", s.DegreesOfFreedom())
	}
	cu, err := c.Curve()
	if err != nil {
		t.Fatalf("Curve: %v", err)
	}
	if math.Abs(cu.Radius-5) > 1e-9 || cu.Center != gmath.P3(1, 2, 3) {
		t.Errorf("kernel circle = center %v r %v, want (1,2,3)/5", cu.Center, cu.Radius)
	}
}

// TestArc3D checks a 3D arc's radius derives from center-to-start and its kernel curve
// passes through the start and end points.
func TestArc3D(t *testing.T) {
	s := NewSketches3D().Add()
	// Quarter arc in the XY plane: center origin, start (1,0,0), end (0,1,0).
	a := s.AddArc3D(gmath.P3(0, 0, 0), gmath.P3(1, 0, 0), gmath.P3(0, 1, 0), true)

	if math.Abs(float64(a.Radius())-1) > 1e-9 {
		t.Errorf("Radius = %v, want 1", a.Radius())
	}
	if s.DegreesOfFreedom() != 9 {
		t.Errorf("a free 3D arc has 9 DOF (3 points), got %d", s.DegreesOfFreedom())
	}
	cu, err := a.Curve()
	if err != nil {
		t.Fatalf("Curve: %v", err)
	}
	start := cu.PointAt(0)
	end := cu.PointAt(1)
	if start.DistanceTo(gmath.P3(1, 0, 0)) > 1e-6 || end.DistanceTo(gmath.P3(0, 1, 0)) > 1e-6 {
		t.Errorf("arc endpoints %v..%v, want (1,0,0)..(0,1,0)", start, end)
	}
}

// TestLine3DAccessorsAndArcAntipodal covers the endpoint accessors and the arc's
// antipodal (degenerate) midpoint branch.
func TestLine3DAccessorsAndArcAntipodal(t *testing.T) {
	s := NewSketches3D().Add()
	l := s.AddLine3D(gmath.P3(0, 0, 0), gmath.P3(1, 0, 0))
	if l.StartPoint() != l.A || l.EndPoint() != l.B {
		t.Error("Start/EndPoint accessors wrong")
	}
	// Start and end antipodal through the center ⇒ the three points are collinear, so the
	// kernel three-point arc construction fails honestly.
	a := s.AddArc3D(gmath.P3(0, 0, 0), gmath.P3(1, 0, 0), gmath.P3(-1, 0, 0), true)
	if _, err := a.Curve(); err == nil {
		t.Error("an antipodal (collinear) arc should fail to build a kernel curve")
	}
}

// TestEntity3DCodecErrors covers the missing-codec, bad-axis, unknown-kind and
// unknown-point error paths of the curve-entity codec.
func TestEntity3DCodecErrors(t *testing.T) {
	if _, err := serializeEntity3D(NewPoint3D(gmath.P3(0, 0, 0))); err == nil {
		t.Error("serializeEntity3D should reject a kind it has no codec for")
	}
	s := NewSketches3D().Add()
	idmap := map[int]*Point3D{1: s.newPoint3D(gmath.P3(0, 0, 0))}
	if err := restoreEntity3D(s, Entity3DData{Kind: "circle", Points: []int{1}, Axis: [3]float64{0, 0, 0}}, idmap); err == nil {
		t.Error("a zero circle axis should fail to restore")
	}
	if err := restoreEntity3D(s, Entity3DData{Kind: "bogus", Points: []int{1}}, idmap); err == nil {
		t.Error("an unknown entity kind should fail to restore")
	}
	if err := restoreEntity3D(s, Entity3DData{Kind: "line", Points: []int{1, 99}}, idmap); err == nil {
		t.Error("a line referencing an unknown point should fail to restore")
	}
}

// TestCurveEntities3DRoundTrip checks line/circle/arc survive marshal→apply with equal
// geometry (positions, radius, axis, construction flag).
func TestCurveEntities3DRoundTrip(t *testing.T) {
	src := NewSketches3D()
	s := src.Add()
	l := s.AddLine3D(gmath.P3(0, 0, 0), gmath.P3(1, 1, 1))
	l.SetConstruction(true)
	s.AddCircle3D(gmath.P3(2, 0, 0), zAxis(), 3)
	s.AddArc3D(gmath.P3(0, 0, 0), gmath.P3(1, 0, 0), gmath.P3(0, 1, 0), true)

	data, err := src.MarshalRecipe3D()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	dst := NewSketches3D()
	if err := dst.ApplyRecipe3D(data); err != nil {
		t.Fatalf("apply: %v", err)
	}
	got := dst.Item(0)
	if got.EntityCount() != 3 {
		t.Fatalf("restored entities = %d, want 3", got.EntityCount())
	}
	ents := got.Entities()
	rl, ok := ents[0].(*Line3D)
	if !ok || !rl.IsConstruction() || rl.B.Position() != gmath.P3(1, 1, 1) {
		t.Errorf("restored line wrong: %+v", ents[0])
	}
	rc, ok := ents[1].(*Circle3D)
	if !ok || float64(rc.Radius) != 3 || rc.Center.Position() != gmath.P3(2, 0, 0) {
		t.Errorf("restored circle wrong: %+v", ents[1])
	}
	ra, ok := ents[2].(*Arc3D)
	if !ok || !ra.CounterClockwise || ra.End.Position() != gmath.P3(0, 1, 0) {
		t.Errorf("restored arc wrong: %+v", ents[2])
	}
}
