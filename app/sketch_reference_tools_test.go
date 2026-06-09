// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
)

// boxForRefs builds a 2×2×3 extruded box on the part so its edges/vertices/faces carry
// reference keys to project/include/derive from.
func boxForRefs(t *testing.T, def *compdef.PartComponentDefinition) {
	t.Helper()
	sk := def.Sketches().Add(sketch.XYPlane())
	c0 := sk.Points().Add(math.P2(-1, -1))
	c1 := sk.Points().Add(math.P2(1, -1))
	c2 := sk.Points().Add(math.P2(1, 1))
	c3 := sk.Points().Add(math.P2(-1, 1))
	sk.Lines().Add(c0, c1)
	sk.Lines().Add(c1, c2)
	sk.Lines().Add(c2, c3)
	sk.Lines().Add(c3, c0)
	feature.NewExtrudeFeatures(def.Features()).AddByDistanceExtent(sk, 0, ops.NewBody, func() float64 { return 3 })
	def.Recompute()
}

// TestProjectGeometryToolEndToEnd drives the 2D Project Geometry tool: while editing a
// sketch, pick a box edge and a box vertex and OK — both become reference geometry on the
// active 2D sketch.
func TestProjectGeometryToolEndToEnd(t *testing.T) {
	s, def := emptyPartSession(t)
	boxForRefs(t, def)
	body := def.SurfaceBodies().All()[0]

	if _, err := s.CreateSketch(sketch.XYPlane()); err != nil {
		t.Fatalf("CreateSketch: %v", err)
	}
	tool := NewProjectGeometryTool()
	s.StartTool(tool)
	tool.Pick(s, EdgeHandle{Edge: body.Edges()[0]})
	tool.Pick(s, VertexHandle{Vertex: body.Vertices()[0]})
	if !tool.CanCommit() {
		t.Fatal("project tool should be ready after an edge + a vertex")
	}
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}

	sk := s.ActiveSketch()
	if got := projectedCount(sk); got != 2 {
		t.Fatalf("active sketch has %d projected entities, want 2", got)
	}
}

// projectedCount counts reference (projected) entities in a 2D sketch.
func projectedCount(sk *sketch.Sketch) int {
	n := 0
	for _, e := range sk.Entities() {
		switch e.(type) {
		case *sketch.ProjectedCurve, *sketch.ProjectedPoint:
			n++
		}
	}
	return n
}

// TestIncludeGeometry3DToolEndToEnd drives the 3D Include Geometry tool: while editing a 3D
// sketch, pick a box edge and vertex and OK — both become included reference geometry.
func TestIncludeGeometry3DToolEndToEnd(t *testing.T) {
	s, def := emptyPartSession(t)
	boxForRefs(t, def)
	body := def.SurfaceBodies().All()[0]

	if _, err := s.CreateSketch3D(); err != nil {
		t.Fatalf("CreateSketch3D: %v", err)
	}
	tool := NewIncludeGeometry3DTool()
	s.StartTool(tool)
	tool.Pick(s, EdgeHandle{Edge: body.Edges()[0]})
	tool.Pick(s, VertexHandle{Vertex: body.Vertices()[0]})
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}

	sk := def.Sketches3D().Item(0)
	var curves, points int
	for _, e := range sk.Entities() {
		switch e.(type) {
		case *sketch.IncludedCurve3D:
			curves++
		case *sketch.IncludedPoint3D:
			points++
		}
	}
	if curves != 1 || points != 1 {
		t.Fatalf("included %d curves / %d points, want 1 / 1", curves, points)
	}
}

// TestSurfaceCurve3DToolEndToEnd drives the Surface Curve tool: pick two box faces and OK
// for an intersection curve, then one face for a silhouette.
func TestSurfaceCurve3DToolEndToEnd(t *testing.T) {
	s, def := emptyPartSession(t)
	boxForRefs(t, def)
	faces := def.SurfaceBodies().All()[0].Faces()
	if len(faces) < 2 {
		t.Fatalf("box has %d faces, want >= 2", len(faces))
	}

	if _, err := s.CreateSketch3D(); err != nil {
		t.Fatalf("CreateSketch3D: %v", err)
	}

	// Intersection of two adjacent faces.
	isect := NewSurfaceCurve3DTool()
	s.StartTool(isect)
	isect.Pick(s, FaceHandle{Face: faces[0]})
	isect.Pick(s, FaceHandle{Face: faces[1]})
	if !isect.CanCommit() {
		t.Fatal("intersection tool needs 2 faces to commit")
	}
	if err := s.OK(); err != nil {
		t.Fatalf("OK intersection: %v", err)
	}

	// Silhouette of one face.
	silh := NewSurfaceCurve3DTool()
	silh.SetSilhouette(true)
	s.StartTool(silh)
	silh.Pick(s, FaceHandle{Face: faces[0]})
	if !silh.CanCommit() {
		t.Fatal("silhouette tool needs 1 face to commit")
	}
	if err := s.OK(); err != nil {
		t.Fatalf("OK silhouette: %v", err)
	}

	sk := def.Sketches3D().Item(0)
	if sk.EntityCount() != 2 {
		t.Fatalf("3D sketch has %d entities, want 2 (intersection + silhouette)", sk.EntityCount())
	}
}

// TestReferenceToolCommands checks the ribbon commands start the matching tools (the
// "Project Geometry" 2D button and the 3D "Include Geometry" / "Surface Curve" buttons).
func TestReferenceToolCommands(t *testing.T) {
	s, _ := emptyPartSession(t)
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("register: %v", err)
	}

	if _, err := s.CreateSketch(sketch.XYPlane()); err != nil {
		t.Fatalf("CreateSketch: %v", err)
	}
	if err := s.Execute("Sketch.Project"); err != nil {
		t.Fatalf("Sketch.Project: %v", err)
	}
	if _, ok := s.ActiveTool().Tool().(*ProjectGeometryTool); !ok {
		t.Fatalf("Sketch.Project started %T, want *ProjectGeometryTool", s.ActiveTool().Tool())
	}
	if err := s.FinishSketch(); err != nil {
		t.Fatalf("FinishSketch: %v", err)
	}

	if _, err := s.CreateSketch3D(); err != nil {
		t.Fatalf("CreateSketch3D: %v", err)
	}
	if err := s.Execute("Sketch3D.Include"); err != nil {
		t.Fatalf("Sketch3D.Include: %v", err)
	}
	if _, ok := s.ActiveTool().Tool().(*IncludeGeometry3DTool); !ok {
		t.Fatalf("Sketch3D.Include started %T, want *IncludeGeometry3DTool", s.ActiveTool().Tool())
	}
	if err := s.Execute("Sketch3D.SurfaceCurve"); err != nil {
		t.Fatalf("Sketch3D.SurfaceCurve: %v", err)
	}
	if _, ok := s.ActiveTool().Tool().(*SurfaceCurve3DTool); !ok {
		t.Fatalf("Sketch3D.SurfaceCurve started %T, want *SurfaceCurve3DTool", s.ActiveTool().Tool())
	}
}

// TestReferenceToolsGuards checks each reference tool refuses to commit before input and
// errors outside the matching sketch environment.
func TestReferenceToolsGuards(t *testing.T) {
	s, _ := emptyPartSession(t)
	proj, inc, surf := NewProjectGeometryTool(), NewIncludeGeometry3DTool(), NewSurfaceCurve3DTool()
	for _, tl := range []Tool{proj, inc, surf} {
		if tl.Name() == "" {
			t.Errorf("%T has no name", tl)
		}
		tl.Start(s)
		tl.Pick(s, nil)
		tl.Cancel(s)
		if tl.CanCommit() {
			t.Errorf("%T should not be committable before input", tl)
		}
		if err := tl.Commit(s); err == nil {
			t.Errorf("%T should refuse to commit with no active sketch/input", tl)
		}
	}
}
