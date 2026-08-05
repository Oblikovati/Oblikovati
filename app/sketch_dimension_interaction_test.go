// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
	"oblikovati.org/renderer"
)

// sketchToScreen projects a sketch-plane point to viewport pixels — the inverse of
// screenToSketch — so a test can aim a click at a computed label anchor.
func sketchToScreen(s *Session, p math.Point2) (float64, float64, bool) {
	return renderer.Project(s.camera, regionNear, regionFar, s.activeSketch.Plane().ToModel(p))
}

// Regression guards for #2017: a sketch dimension could not be selected, deleted or moved,
// because nothing picked one and nothing read the TextPoint the model had stored since #1875.

// dimensionedSketch returns a session editing a sketch holding one 4 cm distance dimension
// across a horizontal line, plus that dimension.
func dimensionedSketch(t *testing.T) (*Session, *sketch.Sketch, *sketch.DimensionConstraint) {
	t.Helper()
	s, sk := sketchSession(t)
	l := sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(4, 0))
	d, err := sk.DimensionConstraints().AddDistance(l.A, l.B, "4 cm")
	if err != nil {
		t.Fatalf("AddDistance: %v", err)
	}
	return s, sk, d
}

// labelOf returns where the session would draw d's label right now.
func labelOf(t *testing.T, s *Session, d *sketch.DimensionConstraint) math.Point2 {
	t.Helper()
	for _, v := range s.SketchDimensions() {
		if v.Dim == d {
			return v.LabelAt
		}
	}
	t.Fatalf("dimension has no view")
	return math.Point2{}
}

// TestStoredTextPointMovesTheLabel is the core regression: TextPoint was persisted, copied and
// settable over the wire, but the view recomputed the anchor every frame and ignored it, so a
// stored placement never reached the screen.
func TestStoredTextPointMovesTheLabel(t *testing.T) {
	s, _, d := dimensionedSketch(t)
	before := labelOf(t, s, d)
	want := math.P2(2, 5)
	d.SetTextPoint(want)
	got := labelOf(t, s, d)
	if got == before {
		t.Fatalf("label ignored the stored TextPoint: still at %v", got)
	}
	if got != want {
		t.Fatalf("label at %v, want the stored TextPoint %v", got, want)
	}
}

// TestMovedDistanceDimensionKeepsItsLineAttached: dragging the text must carry the dimension line
// with it (perpendicularly), or the label floats away from the glyph it belongs to. The witness
// lines still start on the measured points, so the dimension stays attached to its geometry.
func TestMovedDistanceDimensionKeepsItsLineAttached(t *testing.T) {
	s, _, d := dimensionedSketch(t)
	d.SetTextPoint(math.P2(2, 5))
	segs := viewOf(t, s, d).Segments
	// Two witness lines, the dimension line, then two barbs per arrowhead.
	if len(segs) != 7 {
		t.Fatalf("want 7 segments (two witness + one dimension line + two arrowheads), got %d", len(segs))
	}
	dimLine := segs[dimLineIndex]
	if dimLine[0].Y != 5 || dimLine[1].Y != 5 {
		t.Fatalf("dimension line did not follow the text to y=5: %v", dimLine)
	}
	if segs[0][0] != (math.P2(0, 0)) || segs[1][0] != (math.P2(4, 0)) {
		t.Fatalf("witness lines left the measured points: %v %v", segs[0], segs[1])
	}
}

// TestSlidingTextAlongTheLineDoesNotMoveIt: only the perpendicular component displaces the
// dimension line. Sliding the text along its own line must not rotate or shift it.
func TestSlidingTextAlongTheLineDoesNotMoveIt(t *testing.T) {
	s, _, d := dimensionedSketch(t)
	d.SetTextPoint(math.P2(2, 3))
	at2 := viewOf(t, s, d).Segments[2]
	d.SetTextPoint(math.P2(99, 3)) // same perpendicular offset, far along the line
	at99 := viewOf(t, s, d).Segments[2]
	if at2 != at99 {
		t.Fatalf("dimension line moved when the text slid along it: %v vs %v", at2, at99)
	}
}

// viewOf returns d's rendered view.
func viewOf(t *testing.T, s *Session, d *sketch.DimensionConstraint) DimensionView {
	t.Helper()
	for _, v := range s.SketchDimensions() {
		if v.Dim == d {
			return v
		}
	}
	t.Fatalf("dimension has no view")
	return DimensionView{}
}

// TestPickSketchDimensionAtGrabsTheLabel: a click on the label anchor picks the dimension, and a
// click well away from it picks nothing.
func TestPickSketchDimensionAtGrabsTheLabel(t *testing.T) {
	s, _, d := dimensionedSketch(t)
	px, py, ok := sketchToScreen(s, labelOf(t, s, d))
	if !ok {
		t.Fatalf("projecting the label anchor failed")
	}
	got, ok := s.PickSketchDimensionAt(px, py)
	if !ok || got != d {
		t.Fatalf("pick at the label anchor got (%v, %v), want the dimension", got, ok)
	}
	if _, ok := s.PickSketchDimensionAt(px+400, py+400); ok {
		t.Fatalf("pick far from any label found a dimension")
	}
}

// TestDeleteSelectedRemovesDimensions is the reported bug: pressing Delete with a dimension
// selected left it in place.
func TestDeleteSelectedRemovesDimensions(t *testing.T) {
	s, sk, d := dimensionedSketch(t)
	s.Select(SketchDimensionHandle{Dim: d})
	if err := s.DeleteSelectedSketchEntities(); err != nil {
		t.Fatalf("DeleteSelectedSketchEntities: %v", err)
	}
	if n := sk.DimensionConstraints().Count(); n != 0 {
		t.Fatalf("dimension survived Delete: %d remain", n)
	}
	if s.Selection().Count() != 0 {
		t.Fatalf("selection not cleared after deleting the dimension")
	}
}

// TestDeleteSelectedStillRemovesEntities guards the pre-existing behaviour (#1232) against the
// dimension branch: geometry deletion must keep working unchanged.
func TestDeleteSelectedStillRemovesEntities(t *testing.T) {
	s, sk, _ := dimensionedSketch(t)
	l := sk.Lines().Item(0)
	s.Select(SketchEntityHandle{Entity: l})
	if err := s.DeleteSelectedSketchEntities(); err != nil {
		t.Fatalf("DeleteSelectedSketchEntities: %v", err)
	}
	if n := sk.Lines().Count(); n != 0 {
		t.Fatalf("line survived Delete: %d remain", n)
	}
}

// TestDimensionDragMovesOnlyTheLabel: the drag stores a new TextPoint and leaves the measured
// geometry untouched — moving an annotation must never move the model.
func TestDimensionDragMovesOnlyTheLabel(t *testing.T) {
	s, sk, d := dimensionedSketch(t)
	l := sk.Lines().Item(0)
	startA, startB := l.A.Position(), l.B.Position()
	from := labelOf(t, s, d)
	px, py, _ := sketchToScreen(s, from)
	if !s.BeginDimensionDrag(px, py, 0) {
		t.Fatalf("BeginDimensionDrag found no label at its own anchor")
	}
	s.UpdateDimensionDrag(px+30, py)
	s.CommitDimensionDrag()
	if labelOf(t, s, d) == from {
		t.Fatalf("label did not move")
	}
	if l.A.Position() != startA || l.B.Position() != startB {
		t.Fatalf("dragging the label moved the geometry: %v %v", l.A.Position(), l.B.Position())
	}
}

// TestDimensionDragSelectsWhatItGrabs: pressing on a label selects it, so a press-release with no
// movement behaves as a plain click.
func TestDimensionDragSelectsWhatItGrabs(t *testing.T) {
	s, _, d := dimensionedSketch(t)
	px, py, _ := sketchToScreen(s, labelOf(t, s, d))
	if !s.BeginDimensionDrag(px, py, 0) {
		t.Fatalf("BeginDimensionDrag found no label")
	}
	sel := s.SelectedSketchDimensions()
	if len(sel) != 1 || sel[0] != d {
		t.Fatalf("grabbing a label selected %d dimensions, want the one grabbed", len(sel))
	}
	if !viewOf(t, s, d).Selected {
		t.Fatalf("selected dimension is not marked Selected, so it would draw unhighlighted")
	}
}

// TestCancelDimensionDragRestoresTheLabel: Escape mid-drag puts the label back.
func TestCancelDimensionDragRestoresTheLabel(t *testing.T) {
	s, _, d := dimensionedSketch(t)
	from := labelOf(t, s, d)
	px, py, _ := sketchToScreen(s, from)
	s.BeginDimensionDrag(px, py, 0)
	s.UpdateDimensionDrag(px+30, py+30)
	s.CancelDimensionDrag()
	if got := labelOf(t, s, d); got != from {
		t.Fatalf("cancelled drag left the label at %v, want %v", got, from)
	}
}

// The placement was originally stored as a plain sketch point, so dragging the measured line left
// the text behind: the dimension's offset from its geometry silently changed, and a line dragged
// far enough flipped the glyph to the other side of itself. The placement is relative now.

// dimensionOffset returns the SIGNED PERPENDICULAR distance from the measured line to the
// dimension line — the invariant a relative placement preserves. Measuring it in x or y instead
// would only hold while the line keeps its original direction.
func dimensionOffset(t *testing.T, s *Session, d *sketch.DimensionConstraint, l *sketch.Line) float64 {
	t.Helper()
	a, b := l.A.Position(), l.B.Position()
	dir := normalize(a.VectorTo(b))
	perp := math.V2(-dir.Y, dir.X)
	return float64(a.VectorTo(viewOf(t, s, d).Segments[2][0]).Dot(perp))
}

// nearlyEqual compares offsets that are recomputed through different float paths.
func nearlyEqual(a, b float64) bool { return stdmath.Abs(a-b) < 1e-9 }

// TestPlacedDimensionFollowsAMovedLine is the reported bug: the label stayed at its old sketch
// position while the line moved out from under it, so the dimension's own offset silently changed.
func TestPlacedDimensionFollowsAMovedLine(t *testing.T) {
	s, sk, d := dimensionedSketch(t)
	l := sk.Lines().Item(0)
	d.SetTextPoint(math.P2(2, 3)) // dragged 3 above the line
	before := dimensionOffset(t, s, d, l)

	l.A.SetPosition(math.P2(0, 5))
	l.B.SetPosition(math.P2(4, 5))

	if got := dimensionOffset(t, s, d, l); !nearlyEqual(got, before) {
		t.Fatalf("dimension offset changed %v→%v when the line moved; the glyph warped", before, got)
	}
	if got := labelOf(t, s, d); !nearlyEqual(float64(got.Y), 8) {
		t.Fatalf("label at %v after moving the line +5, want it carried to y=8", got)
	}
}

// TestPlacedDimensionFollowsARotatedLine: the placement lives in the dimension's own frame, so it
// rotates with the geometry and stays on the same side of it, instead of being left crossing it.
func TestPlacedDimensionFollowsARotatedLine(t *testing.T) {
	s, sk, d := dimensionedSketch(t)
	l := sk.Lines().Item(0)
	d.SetTextPoint(math.P2(2, 3))
	before := dimensionOffset(t, s, d, l)

	l.B.SetPosition(math.P2(0, 4)) // swing the far end: the line is now vertical

	if got := dimensionOffset(t, s, d, l); !nearlyEqual(got, before) {
		t.Fatalf("offset after rotation = %v, want the placed %v carried around", got, before)
	}
}

// TestUnplacedDimensionStillUsesItsDefault: a dimension nobody dragged keeps the derived anchor,
// so the relative model did not change how an untouched dimension looks.
func TestUnplacedDimensionStillUsesItsDefault(t *testing.T) {
	s, sk, d := dimensionedSketch(t)
	if _, placed := d.TextPoint(); placed {
		t.Fatal("a fresh dimension reports a user placement")
	}
	l := sk.Lines().Item(0)
	before := dimensionOffset(t, s, d, l)
	l.A.SetPosition(math.P2(0, 5))
	l.B.SetPosition(math.P2(4, 5))
	if got := dimensionOffset(t, s, d, l); !nearlyEqual(got, before) {
		t.Fatalf("default offset changed %v→%v when the line moved", before, got)
	}
}

// TestTextPointReportsWhereTheTextActuallyIs: the absolute accessor is derived from live geometry,
// so a caller reading it back after the geometry moved gets the current position, not the stale one
// it was set with.
func TestTextPointReportsWhereTheTextActuallyIs(t *testing.T) {
	_, sk, d := dimensionedSketch(t)
	l := sk.Lines().Item(0)
	d.SetTextPoint(math.P2(2, 3))
	l.A.SetPosition(math.P2(0, 5))
	l.B.SetPosition(math.P2(4, 5))
	tp, ok := d.TextPoint()
	if !ok || !nearlyEqual(float64(tp.Y), 8) {
		t.Fatalf("TextPoint = (%v, %v) after the line moved +5, want y=8", tp, ok)
	}
}
