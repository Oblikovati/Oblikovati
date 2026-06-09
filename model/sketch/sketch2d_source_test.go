// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"testing"

	"oblikovati.org/math"
)

// TestIncludeSketch2DPointInto3D proves a 2D sketch point can be included into a 3D sketch:
// its model position is the 2D point lifted through the host plane, it is constrainable,
// and it tracks edits to the source 2D sketch through UpdateIncluded.
func TestIncludeSketch2DPointInto3D(t *testing.T) {
	src := NewSketches().Add(XZPlane()) // sketch X maps to model X, sketch Y maps to model Z
	p2d := src.Points().Add(math.P2(3, 4))

	s3 := NewSketches3D().Add()
	inc := s3.IncludePoint3D(NewSketch2DPointSource(src, p2d.EntityID()))

	if got := inc.Position(); !got.IsEqualTo(math.P3(3, 0, 4), tol) {
		t.Fatalf("included point = %v, want (3,0,4) (XZ-plane lift of (3,4))", got)
	}
	// The anchor is constrainable and the source tracks edits.
	if !inc.IsConstruction() || inc.Anchor() == nil {
		t.Error("included 2D point should be a constrainable reference anchor")
	}
	p2d.SetPosition(math.P2(10, -2))
	s3.UpdateIncluded()
	if got := inc.Position(); !got.IsEqualTo(math.P3(10, 0, -2), tol) {
		t.Errorf("after source edit, included point = %v, want (10,0,-2)", got)
	}
}

// TestIncludeSketch2DCurveInto3D proves a 2D line can be included into a 3D sketch as a
// reference polyline lifted through the host plane.
func TestIncludeSketch2DCurveInto3D(t *testing.T) {
	src := NewSketches().Add(XZPlane())
	a := src.Points().Add(math.P2(0, 0))
	b := src.Points().Add(math.P2(2, 6))
	line := src.Lines().Add(a, b)

	s3 := NewSketches3D().Add()
	inc := s3.IncludeCurve3D(NewSketch2DCurveSource(src, line.EntityID()))

	pts := inc.Points()
	if len(pts) < 2 {
		t.Fatalf("included curve has %d points, want >= 2", len(pts))
	}
	if !pts[0].IsEqualTo(math.P3(0, 0, 0), tol) || !pts[len(pts)-1].IsEqualTo(math.P3(2, 0, 6), tol) {
		t.Errorf("included polyline ends = %v..%v, want (0,0,0)..(2,0,6)", pts[0], pts[len(pts)-1])
	}
}

// TestIncludeSketch2DLostSourceFreezes checks a deleted (unresolvable) 2D source reports
// lost so the include freezes its last geometry rather than dangling.
func TestIncludeSketch2DLostSourceFreezes(t *testing.T) {
	src := NewSketches().Add(XYPlane())
	p2d := src.Points().Add(math.P2(1, 1))
	s3 := NewSketches3D().Add()
	inc := s3.IncludePoint3D(NewSketch2DPointSource(src, p2d.EntityID()))

	// Simulate the source point going away by including a non-existent id.
	gone := s3.IncludePoint3D(NewSketch2DPointSource(src, ID(999999)))
	s3.UpdateIncluded()
	if gone.Linked() {
		t.Error("an unresolvable 2D source should break the include link")
	}
	if !inc.Linked() {
		t.Error("a live 2D source should remain linked")
	}
}
