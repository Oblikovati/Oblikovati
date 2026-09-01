// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/sketch"
)

// stubEdgePicker is a Picker that returns the given B-rep edge for any pixel, so a test can drive
// an in-sketch edge pick without aiming a real ray at a tessellated edge.
type stubEdgePicker struct{ handle EdgeHandle }

func (p stubEdgePicker) Pick(_, _ float64, f *SelectionFilter) (Selectable, bool) {
	if !f.Accepts(SelectEdge) {
		return nil, false
	}
	return p.handle, true
}

// TestProjectGeometryPicksModelEdgeInSketch is the #1496 regression: with a sketch on a block's
// side face being edited, clicking the block's edge in the viewport must reach the 3D hit-test and
// feed the Project Geometry tool — before the fix every in-sketch click went to the 2D sketch-entity
// picker, which knows nothing about B-rep edges, so the tool silently received nothing and committing
// produced no projected curve ("the button does nothing").
func TestProjectGeometryPicksModelEdgeInSketch(t *testing.T) {
	t.Parallel()
	s := extrudedBox(t, 2, 4) // [0,2]x[0,2]x[0,4]
	part, _ := activePart(s)
	body := partBodies(s)()[0]

	plane := sideFaceSketchPlane(t, body)
	sk, err := s.CreateSketch(plane)
	if err != nil {
		t.Fatalf("CreateSketch: %v", err)
	}
	if !s.InSketch() {
		t.Fatal("expected to be editing the new side-face sketch")
	}

	edge := projectableEdge(t, part, body, plane)
	s.SetPicker(stubEdgePicker{handle: EdgeHandle{Edge: edge}})
	s.StartTool(NewProjectGeometryTool())

	before := countProjectedCurves(sk)
	s.Click(120, 120) // in-sketch click: the #1496 fix routes it to the 3D picker, not the 2D one
	tool := s.ActiveTool().Tool().(*ProjectGeometryTool)
	if !tool.CanCommit() {
		t.Fatal("clicking a model edge in-sketch must feed the Project tool (the #1496 routing fix)")
	}
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	curves := projectedCurves(sk)
	if len(curves)-before != 1 {
		t.Fatalf("committing projected %d curves, want 1", len(curves)-before)
	}
	if got := distinctPointCount(projectionRefPoints(curves[len(curves)-1])); got < 2 {
		t.Errorf("projected curve is degenerate (%d distinct points), want a real segment", got)
	}
}

// TestInSketchRoutingOnlyForReferenceTools guards the routing seam: only a tool that declares it
// picks model references (Project Geometry) diverts in-sketch clicks to the 3D picker; a plain
// sketch-entity tool (Trim) keeps using the 2D sketch picker, and so does no-tool selection.
func TestInSketchRoutingOnlyForReferenceTools(t *testing.T) {
	t.Parallel()
	s, _ := emptyPartSession(t)
	if _, err := s.CreateSketch(sketch.XYPlane()); err != nil {
		t.Fatalf("CreateSketch: %v", err)
	}
	if s.toolPicksModelReferences() {
		t.Error("no active tool must not route in-sketch clicks to the 3D picker")
	}
	s.StartTool(NewProjectGeometryTool())
	if !s.toolPicksModelReferences() {
		t.Error("Project Geometry must route in-sketch clicks to the 3D picker (#1496)")
	}
	s.StartTool(NewSketchTrimTool())
	if s.toolPicksModelReferences() {
		t.Error("a sketch-entity tool must keep the 2D sketch picker, not the 3D picker")
	}
}

// sideFaceSketchPlane returns a sketch plane on one of the box's vertical side faces (the #1496
// setting — a sketch hosted on a side rather than the top cap).
func sideFaceSketchPlane(t *testing.T, body *topo.Body) sketch.Plane {
	t.Helper()
	for _, f := range body.Faces() {
		pl, ok := f.Geometry().(geom.Plane)
		if !ok || stdmath.Abs(float64(pl.Normal().Z)) > 0.1 {
			continue // skip the top/bottom caps
		}
		if plane, ok := sketchPlaneFromFace(FaceHandle{Face: f, Body: body}); ok {
			return plane
		}
	}
	t.Fatal("no side face found on the box")
	return sketch.Plane{}
}

// projectableEdge returns a box edge whose projection onto plane is a real segment (≥2 distinct
// sketch points) — i.e. an edge not perpendicular to the sketch plane — so the regression asserts a
// visible projected curve rather than a degenerate point.
func projectableEdge(t *testing.T, part *compdef.PartComponentDefinition, body *topo.Body, plane sketch.Plane) *topo.Edge {
	t.Helper()
	for _, e := range body.Edges() {
		src := compdef.NewEdgeRefSource(part, string(e.ReferenceKey()))
		pts, ok := src.SamplePoints()
		if !ok {
			continue
		}
		projected := make([]math.Point2, 0, len(pts))
		for _, q := range pts {
			projected = append(projected, plane.ToSketch(q))
		}
		if distinctPointCount(projected) >= 2 {
			return e
		}
	}
	t.Fatal("no projectable (non-perpendicular) edge found on the box")
	return nil
}

func projectedCurves(sk *sketch.Sketch) []*sketch.Projection { return sk.Projections() }

func countProjectedCurves(sk *sketch.Sketch) int { return len(sk.Projections()) }

// projectionRefPoints returns a curve projection's reference-entity defining points (ADR-0055
// phase 3): a projected straight edge drives a reference Line whose two endpoints define its segment.
func projectionRefPoints(c *sketch.Projection) []math.Point2 {
	pts := sketch.DefiningPoints(c.Entity())
	out := make([]math.Point2, len(pts))
	for i, p := range pts {
		out[i] = p.Position()
	}
	return out
}

func distinctPointCount(pts []math.Point2) int {
	if len(pts) == 0 {
		return 0
	}
	n := 1
	for i := 1; i < len(pts); i++ {
		if !pts[i].IsEqualTo(pts[i-1], 1e-9) {
			n++
		}
	}
	return n
}
