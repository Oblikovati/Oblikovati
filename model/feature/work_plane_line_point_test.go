// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"oblikovati.org/math"
)

// AddByLineAndPoint (#1843): the plane contains the line and passes through the point; it is
// degenerate (unhealthy) when the point lies on the line.

// TestAddByLineAndPointBuildsPlane: a plane through the origin X axis and the point (0,5,0) is the
// XY plane — normal ±Z, containing both the axis origin and the point.
func TestAddByLineAndPointBuildsPlane(t *testing.T) {
	g := NewWorkGeometry()
	pt := g.WorkPoints().AddByPosition(func() math.Point3 { return math.P3(0, 5, 0) })
	wp := g.WorkPlanes().AddByLineAndPoint(OriginXAxis, pt.Key())
	if !wp.Health().OK() {
		t.Fatalf("line-and-point plane sick: %+v", wp.Health())
	}
	if !wp.Plane().Normal().AsVector().IsParallelTo(math.V3(0, 0, 1), wtol) {
		t.Errorf("normal = %v, want parallel to +Z (the XY plane)", wp.Plane().Normal())
	}
	// The plane must contain the through-point: its signed distance to the plane is ~0.
	n := wp.Plane().Normal().AsVector()
	if d := n.Dot(wp.Plane().Origin().VectorTo(math.P3(0, 5, 0))); d > wtol || d < -wtol {
		t.Errorf("through-point off the plane by %v", d)
	}
}

// TestAddByLineAndPointDegenerate: a point on the line gives no unique plane (unhealthy).
func TestAddByLineAndPointDegenerate(t *testing.T) {
	g := NewWorkGeometry()
	pt := g.WorkPoints().AddByPosition(func() math.Point3 { return math.P3(3, 0, 0) }) // on the X axis
	wp := g.WorkPlanes().AddByLineAndPoint(OriginXAxis, pt.Key())
	if wp.Health().OK() {
		t.Error("a point lying on the line should make the plane degenerate/unhealthy")
	}
}

// TestWorkPointVisibility: a new work point defaults visible and SetVisible toggles it (#1856).
func TestWorkPointVisibility(t *testing.T) {
	g := NewWorkGeometry()
	pt := g.WorkPoints().AddByPosition(func() math.Point3 { return math.P3(1, 2, 3) })
	if !pt.Visible() {
		t.Error("a new work point should default visible")
	}
	pt.SetVisible(false)
	if pt.Visible() {
		t.Error("SetVisible(false) should hide the work point")
	}
}
