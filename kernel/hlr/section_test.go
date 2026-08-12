// SPDX-License-Identifier: GPL-2.0-only

package hlr

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/subd"
	"oblikovati.org/kernel/topo"
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
			if stdmath.Abs(float64(p.X)) > 1.001 || stdmath.Abs(float64(p.Y)) > 1.001 {
				t.Fatalf("segment point %v outside the cross-section — near half not clipped", p)
			}
		}
	}
}

// lPrism builds an L-section prism: the L profile (0,0)-(3,0)-(3,1)-(1,1)-(1,2)-(0,2) in the XZ
// plane, extruded along +Y by depth. It is deliberately asymmetric along X — the x<1 part is
// twice as tall (z up to 2) as the x>1 part (z up to 1) — so a section cut across X keeps
// different material on each side, which a plain box (symmetric in the view plane) cannot show.
func lPrism(depth float64) *topo.Body {
	prof := [][2]float64{{0, 0}, {3, 0}, {3, 1}, {1, 1}, {1, 2}, {0, 2}}
	n := len(prof)
	verts := make([]math.Point3, 0, 2*n)
	for _, p := range prof { // bottom cap y=0
		verts = append(verts, math.P3(p[0], 0, p[1]))
	}
	for _, p := range prof { // top cap y=depth
		verts = append(verts, math.P3(p[0], depth, p[1]))
	}
	faces := [][]int{reversedRing(n), forwardRing(n, n)} // −Y and +Y caps
	for i := 0; i < n; i++ {
		j := (i + 1) % n
		faces = append(faces, []int{i, n + i, n + j, j}) // outward side quad
	}
	return subd.ToBody(subd.Mesh{Verts: verts, Faces: faces}, "lprism")
}

// forwardRing lists offset..offset+n-1 (a cap wound one way); reversedRing lists them reversed
// (the opposite cap, wound for the other outward normal).
func forwardRing(n, offset int) []int {
	r := make([]int, n)
	for i := range r {
		r[i] = offset + i
	}
	return r
}

func reversedRing(n int) []int {
	r := make([]int, n)
	for i := range r {
		r[i] = n - 1 - i
	}
	return r
}

// TestSectionReverseKeepsOppositeHalf cuts the L-prism at x=1.5 (looking along +X) forward and
// reversed. The forward cut keeps the low x>1.5 material (projected height ≤ 0); the reversed cut
// keeps the tall x<1.5 material (projected height up to +1). The retained curve sets therefore
// differ measurably in vertical extent, proving the opposite half is kept (#1982).
func TestSectionReverseKeepsOppositeHalf(t *testing.T) {
	b := lPrism(2)
	// Plane through (1.5,1,1) normal +X; screen X = +Y, screen Y = +Z (so 2D Y tracks model Z−1).
	view := NewView(math.P3(1.5, 1, 1), math.V3(1, 0, 0), math.V3(0, 0, 1))
	fwd := maxCurveY(ProjectSectionOpts(b, view, ops.DefaultQuality(), SectionOptions{}))
	rev := maxCurveY(ProjectSectionOpts(b, view, ops.DefaultQuality(), SectionOptions{Reverse: true}))
	// Forward keeps only the short flange (model z≤1 ⇒ 2D Y≤0); reverse keeps the tall web
	// (model z up to 2 ⇒ 2D Y up to +1). A symmetric box would give fwd==rev.
	if rev <= fwd+0.5 {
		t.Fatalf("reverse retained-extent %.3f not clearly above forward %.3f — opposite half not kept", rev, fwd)
	}
}

// maxCurveY is the largest screen-Y over all segment endpoints.
func maxCurveY(segs []Segment) float64 {
	m := stdmath.Inf(-1)
	for _, s := range segs {
		m = stdmath.Max(m, stdmath.Max(float64(s.A.Y), float64(s.B.Y)))
	}
	return m
}

// TestSectionLimitedDepthClipsFarGeometry cuts a 2×2×2 box at x=1 (keep x≥1). A full-depth cut
// retains the far face at x=2, whose four edges project to the cross-section square; a cut limited
// to depth 0.5 keeps only x∈[1,1.5], so that far face is clipped away and far fewer edge segments
// survive — while the cut outline and hatch (fixed by the plane) are unchanged (#1982).
func TestSectionLimitedDepthClipsFarGeometry(t *testing.T) {
	b := box(2, 2, 2)
	view := NewView(math.P3(1, 1, 1), math.V3(1, 0, 0), math.V3(0, 0, 1))
	q := ops.DefaultQuality()
	full := ProjectSectionOpts(b, view, q, SectionOptions{})
	limited := ProjectSectionOpts(b, view, q, SectionOptions{Depth: 0.5})

	edgeFull, cutFull, _ := kindCounts(full)
	edgeLim, cutLim, hatchLim := kindCounts(limited)
	if edgeFull == 0 {
		t.Fatal("full-depth section has no retained edges to clip")
	}
	if edgeLim >= edgeFull {
		t.Errorf("limited-depth edges = %d, want fewer than full-depth %d (far face not clipped)", edgeLim, edgeFull)
	}
	if cutLim != cutFull || cutLim == 0 {
		t.Errorf("cut outline changed by depth limit: %d vs %d (want equal, non-zero)", cutLim, cutFull)
	}
	if hatchLim == 0 {
		t.Error("limited-depth section lost its hatch fill (the cut plane is unchanged)")
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
