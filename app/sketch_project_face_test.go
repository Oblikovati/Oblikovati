// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
)

// stubFacePicker is a Picker that returns the given B-rep face for any pixel, so a test can drive
// an in-sketch face pick without aiming a ray at a tessellated face. It honours the installed
// filter, so it also guards that Project Geometry now accepts SelectFace (#2158): before the fix
// the filter rejected faces, the picker returned nothing, and the tool silently did nothing.
type stubFacePicker struct{ handle FaceHandle }

func (p stubFacePicker) Pick(_, _ float64, f *SelectionFilter) (Selectable, bool) {
	if !f.Accepts(SelectFace) {
		return nil, false
	}
	return p.handle, true
}

// buildCylinderPart extrudes a radius-r circle to height h into def, yielding a cylinder whose
// planar end caps are each bounded by a single circular edge — the #2158 geometry (a planar
// circular face whose perimeter must project).
func buildCylinderPart(def *compdef.PartComponentDefinition, r, h float64) {
	sk := def.Sketches().Add(sketch.XYPlane())
	sk.Circles().AddByCenterRadius(math.P2(0, 0), math.Scalar(r))
	feature.NewExtrudeFeatures(def.Features()).AddByDistanceExtent(sk, 0, ops.NewBody, func() float64 { return h })
	def.Recompute()
}

// planarCapFace returns the body's planar face whose outward normal points along +Z (the top cap),
// the planar circular face the user hovers in #2158.
func planarCapFace(t *testing.T, body *topo.Body) *topo.Face {
	t.Helper()
	for _, f := range body.Faces() {
		if pl, ok := f.Geometry().(geom.Plane); ok && float64(pl.Normal().Z) > 0.9 {
			return f
		}
	}
	t.Fatal("no +Z planar cap face found on the cylinder")
	return nil
}

// TestProjectGeometryProjectsCircularCapPerimeter is the #2158 regression: hovering a cylinder's
// planar circular end face during Project Geometry must feed the tool (it accepts faces now), and
// committing must project that face's perimeter — one projected curve, a real circle. Before the
// fix the tool accepted only edges/vertices, so the face never highlighted and nothing projected.
func TestProjectGeometryProjectsCircularCapPerimeter(t *testing.T) {
	s, def := emptyPartSession(t)
	buildCylinderPart(def, 2, 4)
	body := def.SurfaceBodies().All()[0]
	cap := planarCapFace(t, body)

	sk, err := s.CreateSketch(sketch.XYPlane())
	if err != nil {
		t.Fatalf("CreateSketch: %v", err)
	}
	s.SetPicker(stubFacePicker{handle: FaceHandle{Face: cap, Body: body}})
	s.StartTool(NewProjectGeometryTool())

	before := countProjectedCurves(sk)
	s.Click(100, 100)
	tool := s.ActiveTool().Tool().(*ProjectGeometryTool)
	if !tool.CanCommit() {
		t.Fatal("clicking a planar circular face must feed the Project tool (#2158 face acceptance)")
	}
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	curves := projectedCurves(sk)
	if len(curves)-before != 1 {
		t.Fatalf("projecting the circular cap made %d curves, want 1 (its perimeter)", len(curves)-before)
	}
	// The cap perimeter is a circle, so its projection is now a concrete reference *Circle (ADR-0055
	// phase 3): a real circle is a positive radius, not "3 distinct points". A non-analytic perimeter
	// would fall back to a reference spline, checked by its distinct-point count instead.
	last := curves[len(curves)-1].Entity()
	if c, ok := last.(*sketch.Circle); ok {
		if float64(c.Radius) <= 0 {
			t.Errorf("projected cap perimeter has non-positive radius %g, want a real circle", float64(c.Radius))
		}
	} else if got := distinctPointCount(projectionRefPoints(curves[len(curves)-1])); got < 3 {
		t.Errorf("projected cap perimeter is degenerate (%T, %d distinct points), want a real circle", last, got)
	}
}

// TestProjectGeometryProjectsOnPickWithoutOK pins the per-click behaviour: Project Geometry has no
// dialog, so a pick projects immediately (no OK step) and the tool stays armed for the next pick —
// the reported gap where a click on a highlighted face produced no projected geometry.
func TestProjectGeometryProjectsOnPickWithoutOK(t *testing.T) {
	s, def := emptyPartSession(t)
	buildCylinderPart(def, 2, 4)
	body := def.SurfaceBodies().All()[0]
	cap := planarCapFace(t, body)
	sk, err := s.CreateSketch(sketch.XYPlane())
	if err != nil {
		t.Fatalf("CreateSketch: %v", err)
	}
	tool := NewProjectGeometryTool()
	s.StartTool(tool)

	before := countProjectedCurves(sk)
	tool.Pick(s, FaceHandle{Face: cap, Body: body})
	if got := countProjectedCurves(sk) - before; got != 1 {
		t.Fatalf("a face pick projected %d curves before any OK, want 1 (per-click, no dialog)", got)
	}
	if s.ActiveTool() == nil {
		t.Error("Project Geometry deactivated after one pick; it must stay armed for the next (Inventor)")
	}
}

// TestProjectGeometryProjectsAllFaceEdges locks the face→boundary enumeration: projecting a box
// face must project every one of its bounding edges, not just one, so a polygonal face yields its
// whole outline (#2158).
func TestProjectGeometryProjectsAllFaceEdges(t *testing.T) {
	s := extrudedBox(t, 2, 4) // box [0,2]×[0,2]×[0,4]
	body := partBodies(s)()[0]

	cap := planarCapFace(t, body) // top cap: a quad, four bounding edges
	plane := sideFaceSketchPlane(t, body)
	sk, err := s.CreateSketch(plane)
	if err != nil {
		t.Fatalf("CreateSketch: %v", err)
	}

	s.SetPicker(stubFacePicker{handle: FaceHandle{Face: cap, Body: body}})
	s.StartTool(NewProjectGeometryTool())
	before := countProjectedCurves(sk)
	s.Click(100, 100)
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	want := len(cap.Edges())
	if got := countProjectedCurves(sk) - before; got != want {
		t.Fatalf("projecting a %d-edge face made %d curves, want %d (all boundary edges)", want, got, want)
	}
	if want < 3 {
		t.Fatalf("cap face has only %d edges; expected a polygonal cap", want)
	}
}
