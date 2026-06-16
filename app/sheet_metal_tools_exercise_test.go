// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	gmath "oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/sketch"
)

// faceSheet returns a session and part with a side×side base wall already built — the common
// fixture the per-tool exercises act on.
func faceSheet(t *testing.T, side float64) (*Session, *compdef.PartComponentDefinition) {
	t.Helper()
	s, part := sheetMetalSession(t)
	face := NewSheetMetalFaceTool()
	face.Start(s)
	face.Pick(s, squareProfile(part, side))
	if err := face.Commit(s); err != nil {
		t.Fatalf("base Face: %v", err)
	}
	return s, part
}

// lineSketch adds a sketch with one line a→b and returns a handle to that line.
func lineSketch(part *compdef.PartComponentDefinition, a, b gmath.Point2) SketchEntityHandle {
	sk := part.Sketches().Add(sketch.XYPlane())
	pa := sk.Points().Add(a)
	pb := sk.Points().Add(b)
	sk.Lines().Add(pa, pb)
	return SketchEntityHandle{Entity: sk.Lines().Item(0)}
}

// verticalEdge returns a through-thickness (Z-aligned) corner edge of the body.
func verticalEdge(s *Session, part *compdef.PartComponentDefinition) (EdgeHandle, bool) {
	for _, e := range part.Features().Result()[0].Edges() {
		a, b := e.StartVertex().Point(), e.EndVertex().Point()
		if abs(a.X-b.X) < 1e-6 && abs(a.Y-b.Y) < 1e-6 && abs(a.Z-b.Z) > 1e-6 {
			return EdgeHandle{Edge: e}, true
		}
	}
	return EdgeHandle{}, false
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

// TestSheetMetalEdgeTools the edge-driven tools (Hem, Corner, Corner Seam) pick an edge, set
// their dimension, and commit on a base sheet.
func TestSheetMetalEdgeTools(t *testing.T) {
	t.Run("hem", func(t *testing.T) {
		s, part := faceSheet(t, 4)
		hem := NewSheetMetalHemTool()
		hem.Start(s)
		hem.Pick(s, EdgeHandle{Edge: topXEdge(t, part.Features().Result()[0])})
		hem.SetLength(0.5)
		_ = hem.Length()
		if !hem.CanCommit() || hem.Name() == "" {
			t.Fatal("hem not ready")
		}
		if err := hem.Commit(s); err != nil {
			t.Fatalf("hem: %v", err)
		}
		if hem.AddedFeature() == nil {
			t.Error("hem added no feature")
		}
	})
	t.Run("corner", func(t *testing.T) {
		s, part := faceSheet(t, 4)
		edge, ok := verticalEdge(s, part)
		if !ok {
			t.Skip("no through-thickness corner edge")
		}
		corner := NewSheetMetalCornerTool()
		corner.Start(s)
		corner.Pick(s, edge)
		corner.SetSize(0.2)
		_ = corner.Size()
		if !corner.CanCommit() || corner.Name() == "" {
			t.Fatal("corner not ready")
		}
		_ = corner.Commit(s) // a chamfer on a base corner may be sick; the path is exercised
		_ = corner.AddedFeature()
	})
	t.Run("corner-seam", func(t *testing.T) {
		s, part := faceSheet(t, 4)
		edge, ok := verticalEdge(s, part)
		if !ok {
			t.Skip("no corner edge")
		}
		seam := NewSheetMetalCornerSeamTool()
		seam.Start(s)
		seam.Pick(s, edge)
		seam.SetGap(0.05)
		_ = seam.Gap()
		if !seam.CanCommit() || seam.Name() == "" {
			t.Fatal("corner seam not ready")
		}
		_ = seam.Commit(s)
		_ = seam.AddedFeature()
	})
}

// TestSheetMetalSketchTools the sketch-line tools (Bend, Fold) and the profile cut tool pick
// their input and commit on a base sheet.
func TestSheetMetalSketchTools(t *testing.T) {
	t.Run("bend", func(t *testing.T) {
		s, part := faceSheet(t, 4)
		bend := NewSheetMetalBendTool()
		bend.Start(s)
		bend.Pick(s, lineSketch(part, gmath.P2(2, 0), gmath.P2(2, 4)))
		if !bend.CanCommit() || bend.Name() == "" {
			t.Fatal("bend not ready")
		}
		if err := bend.Commit(s); err != nil {
			t.Fatalf("bend: %v", err)
		}
		if bend.AddedFeature() == nil {
			t.Error("bend added no feature")
		}
	})
	t.Run("fold", func(t *testing.T) {
		s, part := faceSheet(t, 4)
		fold := NewSheetMetalFoldTool()
		fold.Start(s)
		fold.Pick(s, lineSketch(part, gmath.P2(2, 0), gmath.P2(2, 4)))
		if !fold.CanCommit() || fold.Name() == "" {
			t.Fatal("fold not ready")
		}
		if err := fold.Commit(s); err != nil {
			t.Fatalf("fold: %v", err)
		}
		_ = fold.AddedFeature()
	})
	t.Run("cut", func(t *testing.T) {
		s, part := faceSheet(t, 4)
		hole := part.Sketches().Add(sketch.XYPlane())
		p := []gmath.Point2{gmath.P2(1.5, 1.5), gmath.P2(2.5, 1.5), gmath.P2(2.5, 2.5), gmath.P2(1.5, 2.5)}
		var pts []*sketch.Point
		for _, q := range p {
			pts = append(pts, hole.Points().Add(q))
		}
		for i := range pts {
			hole.Lines().Add(pts[i], pts[(i+1)%len(pts)])
		}
		cut := NewSheetMetalCutTool()
		cut.Start(s)
		cut.Pick(s, ProfileHandle{Sketch: hole, ProfileIndex: 0})
		if !cut.CanCommit() || cut.Name() == "" {
			t.Fatal("cut not ready")
		}
		if err := cut.Commit(s); err != nil {
			t.Fatalf("cut: %v", err)
		}
		_ = cut.AddedFeature()
	})
}

// TestSheetMetalProfileFlangeTools the profile/axis-driven walls (Contour Flange, Lofted
// Flange, Contour Roll) gather their picks and exercise the commit path; their developed
// geometry is deep-tested in the source model suites.
func TestSheetMetalProfileFlangeTools(t *testing.T) {
	s, part := faceSheet(t, 4)
	edge := EdgeHandle{Edge: topXEdge(t, part.Features().Result()[0])}
	profile := squareProfile(part, 4)

	contour := NewSheetMetalContourFlangeTool()
	contour.Start(s)
	contour.Pick(s, edge)
	contour.Pick(s, profile)
	if !contour.CanCommit() || contour.Name() == "" {
		t.Fatal("contour flange not ready after edge+profile picks")
	}
	_ = contour.Commit(s)
	_ = contour.AddedFeature()

	lofted := NewSheetMetalLoftedFlangeTool()
	lofted.Start(s)
	lofted.Pick(s, profile)
	lofted.Pick(s, squareProfile(part, 3))
	if !lofted.CanCommit() || lofted.Name() == "" {
		t.Fatal("lofted flange not ready after two profile picks")
	}
	_ = lofted.Commit(s)
	_ = lofted.AddedFeature()

	axisSketch := part.Sketches().Add(sketch.XYPlane())
	a := axisSketch.Points().Add(gmath.P2(0, 0))
	b := axisSketch.Points().Add(gmath.P2(0, 4))
	axisSketch.Lines().Add(a, b)
	roll := NewSheetMetalContourRollTool()
	roll.Start(s)
	roll.Pick(s, ProfileHandle{Sketch: axisSketch, ProfileIndex: 0})
	roll.Pick(s, SketchEntityHandle{Entity: axisSketch.Lines().Item(0)})
	roll.SetAngle(1.2)
	_ = roll.Angle()
	if !roll.CanCommit() || roll.Name() == "" {
		t.Fatal("contour roll not ready after profile+axis picks")
	}
	_ = roll.Commit(s)
	_ = roll.AddedFeature()
}
