// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"testing"

	"oblikovati.org/math"
)

// Regression for Oblikovati#2030: a face's holes must come from the arrangement's CLOCKWISE
// (inner-boundary) cycles, not from re-nesting its counter-clockwise ones by containment. The two
// agree only while the regions inside a face are pairwise disjoint; when two of them are ADJACENT,
// containment nesting emits BOTH as holes and the boundary they share is written twice.

// unitSquare returns the segments of an axis-aligned rectangle.
func rectSegments(x0, y0, x1, y1 float64) [][2]math.Point2 {
	c := []math.Point2{math.P2(math.Scalar(x0), math.Scalar(y0)), math.P2(math.Scalar(x1), math.Scalar(y0)),
		math.P2(math.Scalar(x1), math.Scalar(y1)), math.P2(math.Scalar(x0), math.Scalar(y1))}
	out := make([][2]math.Point2, 0, 4)
	for i := range c {
		out = append(out, [2]math.Point2{c[i], c[(i+1)%4]})
	}
	return out
}

// TestArrangeAdjacentRegionsFormOneHole is the reduced form of the defect. Inside a big outer
// square sit two rectangles that SHARE an edge — the arrangement has three bounded regions (each
// rectangle, and the outer frame around them). The frame's inner boundary is ONE loop running
// around the pair; emitting the two rectangles as two separate holes would write their shared edge
// twice, which downstream becomes an edge whose two uses lie on the same face.
func TestArrangeAdjacentRegionsFormOneHole(t *testing.T) {
	t.Parallel()
	var segs [][2]math.Point2
	segs = append(segs, rectSegments(0, 0, 10, 10)...) // outer frame
	segs = append(segs, rectSegments(2, 2, 5, 6)...)   // left cell
	segs = append(segs, rectSegments(5, 2, 8, 6)...)   // right cell, sharing the x=5 edge

	frame := faceWithOuterArea(t, Arrange(segs), 100)
	if len(frame.Holes) != 1 {
		t.Fatalf("frame has %d holes, want 1 — two adjacent regions form ONE connected opening", len(frame.Holes))
	}
	// The single hole must run around the union (2..8 in x, 2..6 in y), not around one cell.
	lo, hi := boundsOf(frame.Holes[0])
	if lo.X != 2 || hi.X != 8 || lo.Y != 2 || hi.Y != 6 {
		t.Errorf("hole spans [%v %v]..[%v %v], want [2 2]..[8 6] (the union of both cells)", lo.X, lo.Y, hi.X, hi.Y)
	}
	// The shared x=5 edge is interior to the opening, so it must not appear on the frame at all.
	for _, p := range frame.Holes[0] {
		if p.X == 5 && p.Y > 2 && p.Y < 6 {
			t.Errorf("the shared edge at x=5 leaked onto the frame's boundary at %v", p)
		}
	}
}

// TestArrangeDisjointRegionsStayTwoHoles pins the other side: two regions that do NOT touch are
// two separate openings, and must stay two holes.
func TestArrangeDisjointRegionsStayTwoHoles(t *testing.T) {
	t.Parallel()
	var segs [][2]math.Point2
	segs = append(segs, rectSegments(0, 0, 10, 10)...)
	segs = append(segs, rectSegments(2, 2, 4, 6)...)
	segs = append(segs, rectSegments(6, 2, 8, 6)...)

	if frame := faceWithOuterArea(t, Arrange(segs), 100); len(frame.Holes) != 2 {
		t.Errorf("frame has %d holes, want 2 — the cells are disjoint openings", len(frame.Holes))
	}
}

// TestArrangeNestedFramesKeepTheirOwnHole pins multi-level nesting: three concentric squares give a
// frame whose hole is the middle square, and a middle frame whose hole is the inner square — each
// hole assigned to its DIRECT parent, not to the outermost face.
func TestArrangeNestedFramesKeepTheirOwnHole(t *testing.T) {
	t.Parallel()
	var segs [][2]math.Point2
	segs = append(segs, rectSegments(0, 0, 10, 10)...)
	segs = append(segs, rectSegments(2, 2, 8, 8)...)
	segs = append(segs, rectSegments(4, 4, 6, 6)...)
	faces := Arrange(segs)

	outer := faceWithOuterArea(t, faces, 100)
	if len(outer.Holes) != 1 {
		t.Fatalf("outer frame has %d holes, want 1", len(outer.Holes))
	}
	if lo, hi := boundsOf(outer.Holes[0]); lo.X != 2 || hi.X != 8 {
		t.Errorf("outer frame's hole spans x %v..%v, want 2..8 (the middle square)", lo.X, hi.X)
	}
	middle := faceWithOuterArea(t, faces, 36)
	if len(middle.Holes) != 1 {
		t.Fatalf("middle frame has %d holes, want 1", len(middle.Holes))
	}
	if lo, hi := boundsOf(middle.Holes[0]); lo.X != 4 || hi.X != 6 {
		t.Errorf("middle frame's hole spans x %v..%v, want 4..6 (the inner square)", lo.X, hi.X)
	}
}

// faceWithOuterArea returns the arranged face whose outer loop encloses the given area.
func faceWithOuterArea(t *testing.T, faces []Face2D, area float64) Face2D {
	t.Helper()
	for _, f := range faces {
		if a := signedArea2D(f.Outer); a > area-1e-6 && a < area+1e-6 {
			return f
		}
	}
	t.Fatalf("no arranged face of area %v (got %d faces)", area, len(faces))
	return Face2D{}
}

// boundsOf returns a loop's axis-aligned bounds.
func boundsOf(loop []math.Point2) (lo, hi math.Point2) {
	lo, hi = loop[0], loop[0]
	for _, p := range loop {
		lo = math.P2(min(lo.X, p.X), min(lo.Y, p.Y))
		hi = math.P2(max(hi.X, p.X), max(hi.Y, p.Y))
	}
	return lo, hi
}
