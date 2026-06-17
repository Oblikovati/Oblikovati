// SPDX-License-Identifier: GPL-2.0-only

package hlr

import (
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
)

// kindCounts tallies segments by kind.
func kindCounts(segs []Segment) (edge, cut, hatch int) {
	for _, s := range segs {
		switch s.Kind {
		case KindCut:
			cut++
		case KindHatch:
			hatch++
		default:
			edge++
		}
	}
	return
}

// TestSectionBoxCutsHatchesAndClips sections a 2×2×2 box (spanning [0,2]³) with the plane x=1
// looking along +X (keep the x≥1 half). The cut cross-section is the 2×2 square at x=1: the
// outline is four bold segments, it is hatched, and — crucially — every projected segment lies
// within the cross-section bounds, proving the near half (x<1) was clipped away rather than
// drawn over.
func TestSectionBoxCutsHatchesAndClips(t *testing.T) {
	b := box(2, 2, 2)
	// Plane through (1,1,1) with normal +X; screen up = +Z, so screen = (worldY, worldZ).
	view := NewView(math.P3(1, 1, 1), math.V3(1, 0, 0), math.V3(0, 0, 1))
	segs := ProjectSection(b, view, ops.DefaultQuality())

	edge, cut, hatch := kindCounts(segs)
	if cut < 4 {
		t.Errorf("cut outline = %d segments, want ≥4 (the square at x=1)", cut)
	}
	if hatch < 10 {
		t.Errorf("hatch = %d lines, want a filled cross-section (≥10)", hatch)
	}
	if edge == 0 {
		t.Error("section has no retained-half edge segments")
	}
	// Every segment must lie within the 2×2 cross-section (screen coords in [-1,1]); a segment
	// outside it would mean the removed near half leaked into the projection.
	for _, s := range segs {
		for _, p := range [2]math.Point2{s.A, s.B} {
			if abs(float64(p.X)) > 1.001 || abs(float64(p.Y)) > 1.001 {
				t.Fatalf("segment point %v outside the cross-section — near half not clipped", p)
			}
		}
	}
}

// TestSectionEdgesAllVisible checks the cut-away half's edges are all visible — with the near
// half removed, nothing occludes the retained geometry from the section viewpoint.
func TestSectionEdgesAllVisible(t *testing.T) {
	b := box(2, 3, 4) // distinct dims
	view := NewView(math.P3(1, 1.5, 2), math.V3(0, 1, 0), math.V3(0, 0, 1))
	segs := ProjectSection(b, view, ops.DefaultQuality())
	for _, s := range segs {
		if s.Kind == KindEdge && !s.Visible {
			t.Errorf("retained-half edge %v→%v hidden, want visible in a cut-away", s.A, s.B)
		}
	}
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
