// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
)

// TestCreateSketchAutoProjectsOrigin: a freshly created sketch carries the projected origin
// centre point at (0,0), the Inventor default the bug report (#1262) asked for.
func TestCreateSketchAutoProjectsOrigin(t *testing.T) {
	s, _ := emptyPartSession(t)
	sk, err := s.CreateSketch(sketch.XYPlane())
	if err != nil {
		t.Fatalf("CreateSketch: %v", err)
	}
	var origins []*sketch.ProjectedPoint
	for _, e := range sk.Entities() {
		if pp, ok := e.(*sketch.ProjectedPoint); ok {
			origins = append(origins, pp)
		}
	}
	if len(origins) != 1 {
		t.Fatalf("new sketch has %d projected points, want 1 (the origin centre)", len(origins))
	}
	if got := origins[0].Position(); !got.IsEqualTo(math.P2(0, 0), 1e-9) {
		t.Errorf("projected origin at %v, want (0,0)", got)
	}
	if origins[0].SourceID() != string(feature.OriginCenter) {
		t.Errorf("projected origin source = %q, want %q", origins[0].SourceID(), feature.OriginCenter)
	}
}

// TestProjectGeometryToolProjectsDatums drives the Project Geometry tool on the part's datum
// geometry — the origin centre point, an origin axis and an origin plane (the references a user
// reaches from the browser tree) — and confirms each becomes reference geometry (#1262). The
// XZ origin plane meets the XY sketch along the X axis, so it projects a line.
func TestProjectGeometryToolProjectsDatums(t *testing.T) {
	s, def := emptyPartSession(t)
	if _, err := s.CreateSketch(sketch.XYPlane()); err != nil {
		t.Fatalf("CreateSketch: %v", err)
	}
	center, _ := def.WorkGeometry().WorkPointByRef(feature.OriginCenter)
	xAxis, _ := def.WorkGeometry().AxisByRef(feature.OriginXAxis)
	xzPlane, _ := def.WorkGeometry().WorkPlaneByRef(feature.OriginXZPlane)

	xyPlane, _ := def.WorkGeometry().WorkPlaneByRef(feature.OriginXYPlane) // parallel to the sketch

	tool := NewProjectGeometryTool()
	s.StartTool(tool)
	tool.Pick(s, WorkPointHandle{Point: center})
	tool.Pick(s, WorkAxisHandle{Axis: xAxis})
	tool.Pick(s, WorkPlaneHandle{Plane: xzPlane})
	tool.Pick(s, WorkPlaneHandle{Plane: xyPlane}) // parallel: must be skipped on commit
	if !tool.CanCommit() {
		t.Fatal("tool should be ready after picking datum geometry")
	}
	if got := len(tool.Picks()); got != 4 {
		t.Errorf("Picks() = %d, want 4 datum picks for the highlight", got)
	}
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}

	sk := s.ActiveSketch()
	var points, curves int
	for _, e := range sk.Entities() {
		switch e.(type) {
		case *sketch.ProjectedPoint:
			points++
		case *sketch.ProjectedCurve:
			curves++
		}
	}
	// 2 points = auto-projected origin (CreateSketch) + the explicitly projected centre point;
	// 2 curves = the projected origin X axis + the XZ-plane∩sketch intersection line.
	if points != 2 || curves != 2 {
		t.Fatalf("projected %d points / %d curves, want 2 / 2", points, curves)
	}
}
