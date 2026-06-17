// SPDX-License-Identifier: GPL-2.0-only

package hlr

import (
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/subd"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// box builds a validated solid box of the given size with its near corner at the origin.
func box(sx, sy, sz float64) *topo.Body {
	return subd.ToBody(subd.Box(sx, sy, sz), "box")
}

// visibleHidden counts how many projected segments are visible vs hidden.
func visibleHidden(segs []Segment) (visible, hidden int) {
	for _, s := range segs {
		if s.Visible {
			visible++
		} else {
			hidden++
		}
	}
	return
}

// TestIsoCubeNineVisibleThreeHidden is the textbook HLR result: a cube viewed from a general
// 3-face direction shows 9 visible edges (the three faces toward the viewer) and 3 hidden
// edges (the three meeting at the far corner, behind the solid). The 12 straight edges each
// project to one segment, so the counts are exact. A general direction (3,4,5) — not the
// perfectly symmetric (1,1,1) isometric, where the far corner projects exactly onto the near
// one and the classification is FP-degenerate — keeps the result platform-stable.
func TestIsoCubeNineVisibleThreeHidden(t *testing.T) {
	b := box(2, 2, 2)
	// Look along +(3,4,5) into the screen: the near corner (0,0,0) faces the viewer, the far
	// corner (2,2,2) is hidden; no two corners project coincident.
	view := NewView(math.P3(1, 1, 1), math.V3(3, 4, 5), math.V3(0, 0, 1))
	segs := Project(b, view, ops.DefaultQuality())

	visible, hidden := visibleHidden(segs)
	if visible != 9 || hidden != 3 {
		t.Fatalf("iso cube = %d visible / %d hidden (%d total), want 9/3", visible, hidden, len(segs))
	}
	// Every segment carries its source edge's reference key (associativity).
	for _, s := range segs {
		if len(s.EdgeKey) == 0 {
			t.Errorf("segment %+v has no edge key", s)
		}
	}
}

// TestFrontViewCubeOutlineVisible checks a front-on cube: the four front-face edges are
// visible and the four back-face edges are hidden (they project coincident with the front).
// The four edges running along the view direction collapse to points and are dropped.
func TestFrontViewCubeOutlineVisible(t *testing.T) {
	b := box(2, 2, 2)
	// Front view: look along +Y, screen X = +X, screen up = +Z.
	view := NewView(math.P3(1, 1, 1), math.V3(0, 1, 0), math.V3(0, 0, 1))
	segs := Project(b, view, ops.DefaultQuality())

	visible, hidden := visibleHidden(segs)
	if len(segs) != 8 {
		t.Fatalf("front cube projected %d segments, want 8 (4 front + 4 back; 4 end-on dropped)", len(segs))
	}
	if visible != 4 || hidden != 4 {
		t.Errorf("front cube = %d visible / %d hidden, want 4/4", visible, hidden)
	}
}
