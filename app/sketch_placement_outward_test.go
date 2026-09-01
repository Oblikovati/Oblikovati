// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// An in-place dimension was drawn ON its witness segment, which for a rectangle IS an edge of the
// shape — so the value box sat on the geometry, often inside it (#2034). Each field now carries
// the direction its dimension stands off in, away from the shape.

// TestEveryFieldPointsAwayFromTheShape is the regression: whichever edge a rectangle's dimension
// measures, its offset direction must lead OUT of the rectangle, never into it.
func TestEveryFieldPointsAwayFromTheShape(t *testing.T) {
	t.Parallel()
	s, _ := sketchSession(t)
	s.StartTool(NewRectangleTool())
	s.Click(40, 40)
	if _, ok := s.CursorSketchPoint(140, 120); !ok {
		t.Fatal("the cursor is not over the sketch plane")
	}

	fields := s.PlacementFields()
	if len(fields) != 2 {
		t.Fatalf("got %d fields, want Width and Height", len(fields))
	}
	r, _ := s.ActiveToolRecipe(s.lastCursorSketchPoint)
	pts, _ := sketch.RecipeOutline(r)
	if len(pts) < 3 {
		t.Fatal("a placed rectangle must enclose an area to measure from")
	}
	var sum math.Vector2
	for _, p := range pts {
		sum = sum.Add(math.V2(p.X, p.Y))
	}
	centroid := math.P2(0, 0).TranslateBy(sum.Scale(1 / float64(len(pts))))
	for _, f := range fields {
		mid := math.P2((f.Witness[0].X+f.Witness[1].X)/2, (f.Witness[0].Y+f.Witness[1].Y)/2)
		if centroid.VectorTo(mid).Dot(f.Outward) <= 0 {
			t.Errorf("%s dimension offsets toward the shape's interior (outward=%v)", f.Label, f.Outward)
		}
	}
}

// TestOutwardIsPerpendicularAndUnit: the head offsets by a pixel gap along this direction, so it
// has to be a unit normal of the witness segment — a non-unit or skewed vector would put the
// dimension line at the wrong distance or angle.
func TestOutwardIsPerpendicularAndUnit(t *testing.T) {
	t.Parallel()
	s, _ := sketchSession(t)
	s.StartTool(NewRectangleTool())
	s.Click(40, 40)
	if _, ok := s.CursorSketchPoint(140, 120); !ok {
		t.Fatal("the cursor is not over the sketch plane")
	}

	fields := s.PlacementFields()
	if len(fields) != 2 {
		t.Fatalf("got %d fields, want 2 — an empty set would pass this test vacuously", len(fields))
	}
	for _, f := range fields {
		if l := float64(f.Outward.Length()); l < 0.999 || l > 1.001 {
			t.Errorf("%s outward length = %v, want 1", f.Label, l)
		}
		edge := f.Witness[0].VectorTo(f.Witness[1])
		if d := float64(edge.Dot(f.Outward)); d > 1e-9 || d < -1e-9 {
			t.Errorf("%s outward is not perpendicular to its witness (dot=%v)", f.Label, d)
		}
	}
}

// TestOpenShapeStillGetsADirection: a shape with no enclosed area — a single rubber-banding line —
// has no inside to point away from, but its dimension still has to stand off the geometry rather
// than land on it.
func TestOpenShapeStillGetsADirection(t *testing.T) {
	t.Parallel()
	s, _ := sketchSession(t)
	s.StartTool(NewLineTool())
	s.Click(40, 40)
	if _, ok := s.CursorSketchPoint(140, 40); !ok {
		t.Fatal("the cursor is not over the sketch plane")
	}

	open := s.PlacementFields()
	if len(open) == 0 {
		t.Fatal("a line being rubber-banded must still offer its Length/Angle fields")
	}
	for _, f := range open {
		if l := float64(f.Outward.Length()); l < 0.999 || l > 1.001 {
			t.Errorf("%s outward length = %v, want a unit direction even with no enclosed area", f.Label, l)
		}
	}
}

// TestCommittedDimensionStandsOutsideTheShape is the reported defect: the in-place dimension
// stood off the rectangle correctly, then the dimension the commit created appeared INSIDE it —
// the annotation jumped the moment the shape was made (#2034).
func TestCommittedDimensionStandsOutsideTheShape(t *testing.T) {
	t.Parallel()
	// BOTH drag directions. The default anchor offsets by a fixed perpendicular handedness, so
	// which side it lands on depends on which way the rectangle was dragged — one direction was
	// already outside by luck, the other put the dimension inside the shape.
	for _, drag := range []struct {
		name           string
		x0, y0, x1, y1 float64
	}{
		{"down-right", 40, 40, 140, 120},
		{"up-right", 40, 120, 140, 40},
	} {
		t.Run(drag.name, func(t *testing.T) { assertDimensionOutside(t, drag.x0, drag.y0, drag.x1, drag.y1) })
	}
}

// assertDimensionOutside places a rectangle with a locked Width and checks the dimension the
// commit created stands clear of it.
func assertDimensionOutside(t *testing.T, x0, y0, x1, y1 float64) {
	t.Helper()
	s, sk := sketchSession(t)
	s.StartTool(NewRectangleTool())
	s.Click(x0, y0)
	for _, r := range "50" {
		s.PlacementFieldInput(r)
	}
	s.PlacementFieldTab() // lock Width so the commit creates a driving dimension
	s.Click(x1, y1)

	dims := sk.DimensionConstraints().All()
	if len(dims) != 1 {
		t.Fatalf("got %d dimensions, want the locked Width", len(dims))
	}
	at, ok := dims[0].LabelAnchor()
	if !ok {
		t.Fatal("the committed dimension has no label anchor to place")
	}
	// Clear of the shape, not merely off it: the default anchor sits ON the measured edge, which
	// a strict inside/outside test would wave through.
	if gap := clearanceOutside(sk, at); gap < 0.3 {
		t.Errorf("the committed dimension sits %v from the rectangle (at %v) — it must stand clear of the shape it measures", gap, at)
	}
}

// clearanceOutside is how far p lies beyond the bounding box of the sketch's lines — 0 when it is
// inside it or on its edge.
func clearanceOutside(sk *sketch.Sketch, p math.Point2) float64 {
	minX, minY := math.Scalar(1e9), math.Scalar(1e9)
	maxX, maxY := math.Scalar(-1e9), math.Scalar(-1e9)
	for i := 0; i < sk.Lines().Count(); i++ {
		l := sk.Lines().Item(i)
		for _, v := range []*sketch.Point{l.A, l.B} {
			minX, maxX = math.Scalar(stdMin(float64(minX), float64(v.X))), math.Scalar(stdMax(float64(maxX), float64(v.X)))
			minY, maxY = math.Scalar(stdMin(float64(minY), float64(v.Y))), math.Scalar(stdMax(float64(maxY), float64(v.Y)))
		}
	}
	dx := stdMax(float64(minX-p.X), float64(p.X-maxX))
	dy := stdMax(float64(minY-p.Y), float64(p.Y-maxY))
	return stdMax(stdMax(dx, dy), 0)
}

func stdMin(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func stdMax(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
