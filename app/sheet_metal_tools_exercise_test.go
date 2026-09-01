// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	stdmath "math"
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
		if stdmath.Abs(a.X-b.X) < 1e-6 && stdmath.Abs(a.Y-b.Y) < 1e-6 && stdmath.Abs(a.Z-b.Z) > 1e-6 {
			return EdgeHandle{Edge: e}, true
		}
	}
	return EdgeHandle{}, false
}

// wantDraftReady asserts a commit-ready tool builds a non-nil draft — the contract the
// sick-config commit gate relies on to never be skipped by omission (#1626, audit I3).
func wantDraftReady(t *testing.T, s *Session, tool DraftPreviewable) {
	t.Helper()
	draft, ok := tool.DraftFeature(s)
	if !ok || draft == nil {
		t.Fatalf("commit-ready tool built no draft (ok=%v, draft=%v)", ok, draft)
	}
}

// TestSheetMetalEdgeTools the edge-driven tools (Hem, Corner, Corner Seam) pick an edge, set
// their dimension, and commit on a base sheet.
func TestSheetMetalEdgeTools(t *testing.T) {
	t.Parallel()
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
		wantDraftReady(t, s, hem)
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
		wantDraftReady(t, s, corner)
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
		wantDraftReady(t, s, seam)
		_ = seam.Commit(s)
		_ = seam.AddedFeature()
	})
}

// TestSheetMetalSketchTools the sketch-line tools (Bend, Fold) and the profile cut tool pick
// their input and commit on a base sheet.
func TestSheetMetalSketchTools(t *testing.T) {
	t.Parallel()
	t.Run("bend", func(t *testing.T) {
		s, part := faceSheet(t, 4)
		bend := NewSheetMetalBendTool()
		bend.Start(s)
		bend.Pick(s, lineSketch(part, gmath.P2(2, 0), gmath.P2(2, 4)))
		if !bend.CanCommit() || bend.Name() == "" {
			t.Fatal("bend not ready")
		}
		wantDraftReady(t, s, bend)
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
		wantDraftReady(t, s, fold)
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
		wantDraftReady(t, s, cut)
		if err := cut.Commit(s); err != nil {
			t.Fatalf("cut: %v", err)
		}
		_ = cut.AddedFeature()
	})
}

// TestSheetMetalF03Tools the F03 modify tools (Lip, Rip, Punch, Cosmetic Bend) pick their input
// and commit on a base sheet.
func TestSheetMetalF03Tools(t *testing.T) {
	t.Parallel()
	t.Run("lip", func(t *testing.T) {
		s, part := faceSheet(t, 4)
		lip := NewSheetMetalLipTool()
		lip.Start(s)
		lip.Pick(s, EdgeHandle{Edge: topXEdge(t, part.Features().Result()[0])})
		lip.SetHeight(1.0)
		lip.SetReturnLength(0.4)
		lip.SetAngle(halfPiAngle)
		_, _, _ = lip.Height(), lip.ReturnLength(), lip.Angle()
		if !lip.CanCommit() || lip.Name() == "" {
			t.Fatal("lip not ready")
		}
		if err := lip.Commit(s); err != nil {
			t.Fatalf("lip: %v", err)
		}
		if lip.AddedFeature() == nil {
			t.Error("lip added no feature")
		}
	})
	t.Run("rip", func(t *testing.T) {
		s, part := faceSheet(t, 4)
		rip := NewSheetMetalRipTool()
		rip.Start(s)
		rip.Pick(s, lineSketch(part, gmath.P2(1, 1.5), gmath.P2(3, 1.5))) // partial line
		rip.SetGap(0.05)
		_ = rip.Gap()
		if !rip.CanCommit() || rip.Name() == "" {
			t.Fatal("rip not ready")
		}
		if err := rip.Commit(s); err != nil {
			t.Fatalf("rip: %v", err)
		}
		_ = rip.AddedFeature()
	})
	t.Run("punch", func(t *testing.T) {
		s, part := faceSheet(t, 4)
		holes := part.Sketches().Add(sketch.XYPlane())
		for _, c := range []gmath.Point2{gmath.P2(1, 1), gmath.P2(3, 3)} {
			q := []gmath.Point2{gmath.P2(c.X-0.3, c.Y-0.3), gmath.P2(c.X+0.3, c.Y-0.3), gmath.P2(c.X+0.3, c.Y+0.3), gmath.P2(c.X-0.3, c.Y+0.3)}
			var pts []*sketch.Point
			for _, p := range q {
				pts = append(pts, holes.Points().Add(p))
			}
			for i := range pts {
				holes.Lines().Add(pts[i], pts[(i+1)%len(pts)])
			}
		}
		punch := NewSheetMetalPunchTool()
		punch.Start(s)
		punch.Pick(s, ProfileHandle{Sketch: holes, ProfileIndex: 0})
		if !punch.CanCommit() || punch.Name() == "" {
			t.Fatal("punch not ready")
		}
		if err := punch.Commit(s); err != nil {
			t.Fatalf("punch: %v", err)
		}
		_ = punch.AddedFeature()
	})
	t.Run("cosmetic-bend", func(t *testing.T) {
		s, part := faceSheet(t, 4)
		cb := NewSheetMetalCosmeticBendTool()
		cb.Start(s)
		cb.Pick(s, lineSketch(part, gmath.P2(2, 0), gmath.P2(2, 4)))
		cb.SetAngle(halfPiAngle)
		_ = cb.Angle()
		if !cb.CanCommit() || cb.Name() == "" {
			t.Fatal("cosmetic bend not ready")
		}
		wantDraftReady(t, s, cb)
		if err := cb.Commit(s); err != nil {
			t.Fatalf("cosmetic bend: %v", err)
		}
		_ = cb.AddedFeature()
	})
}

// TestSheetMetalProfileFlangeTools the profile/axis-driven walls (Contour Flange, Lofted
// Flange, Contour Roll) gather their picks and exercise the commit path; their developed
// geometry is deep-tested in the source model suites.
func TestSheetMetalProfileFlangeTools(t *testing.T) {
	t.Parallel()
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
	wantDraftReady(t, s, contour)
	_ = contour.Commit(s)
	_ = contour.AddedFeature()

	lofted := NewSheetMetalLoftedFlangeTool()
	lofted.Start(s)
	lofted.Pick(s, profile)
	lofted.Pick(s, squareProfile(part, 3))
	if !lofted.CanCommit() || lofted.Name() == "" {
		t.Fatal("lofted flange not ready after two profile picks")
	}
	wantDraftReady(t, s, lofted)
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
	wantDraftReady(t, s, roll)
	_ = roll.Commit(s)
	_ = roll.AddedFeature()
}
