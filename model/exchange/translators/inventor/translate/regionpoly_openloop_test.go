// SPDX-License-Identifier: GPL-2.0-only

package translate

import (
	"testing"

	m "oblikovati.org/math"
	"oblikovati.org/model/exchange/translators/inventor/ipt"
)

// line builds a straight region edge between two points.
func line(ax, ay, bx, by float64) ipt.RegionEdge {
	return ipt.RegionEdge{Kind: ipt.EdgeLine, Line: ipt.Line{A: ipt.Point2D{X: ax, Y: ay}, B: ipt.Point2D{X: bx, Y: by}}}
}

// closedTriangle is a material loop that chains to a real face.
func closedTriangle() ipt.RegionLoop {
	return ipt.RegionLoop{Edges: []ipt.RegionEdge{
		line(0, 0, 1, 0), line(1, 0, 0, 1), line(0, 1, 0, 0),
	}}
}

// openSliver is a two-edge chain sharing one vertex whose other ends are far apart: it bounds no
// closed area, the shape TapePath's tape-guide patch carries alongside its real ribbon faces.
func openSliver(cut bool) ipt.RegionLoop {
	return ipt.RegionLoop{Cut: cut, Edges: []ipt.RegionEdge{
		line(0, 0, 0.001, 0.001), line(0, 0, 5, 5),
	}}
}

// TestRegionBoundariesDropsOpenMaterialSliver: a region of one real face plus an open material
// sliver reconstructs the face and drops the sliver (an open curve bounds no material), rather than
// failing the whole region to the curve-set fallback.
func TestRegionBoundariesDropsOpenMaterialSliver(t *testing.T) {
	outers, holes, ok := regionBoundaries([]ipt.RegionLoop{closedTriangle(), openSliver(false)})
	if !ok {
		t.Fatalf("regionBoundaries declined; want it to keep the closed face and drop the open sliver")
	}
	if len(outers) != 1 {
		t.Errorf("outers = %d, want 1 (the triangle; the open sliver bounds no face)", len(outers))
	}
	if len(holes) != 0 {
		t.Errorf("holes = %d, want 0", len(holes))
	}
}

// TestRegionBoundariesDeclinesOnOpenCut: an unreadable CUT loop is NOT dropped — silently losing a
// hole would over-fill the face — so the region declines and keeps the safer fallback.
func TestRegionBoundariesDeclinesOnOpenCut(t *testing.T) {
	if _, _, ok := regionBoundaries([]ipt.RegionLoop{closedTriangle(), openSliver(true)}); ok {
		t.Errorf("regionBoundaries accepted a region with an unreadable cut loop; want decline")
	}
}

// TestSmallestContainingAreaPicksTightestLoop: a point inside two nested outer loops resolves to the
// SMALLER loop's area (a cell there can belong to at most the tighter one), and polygonArea is the
// enclosed area. This backs the containment area-fit guard that keeps a big keep-cell out of a small
// cut loop (FlangeReelMotor).
func TestSmallestContainingAreaPicksTightestLoop(t *testing.T) {
	big := []m.Point2{m.P2(0, 0), m.P2(10, 0), m.P2(10, 10), m.P2(0, 10)} // 100
	small := []m.Point2{m.P2(3, 3), m.P2(7, 3), m.P2(7, 7), m.P2(3, 7)}   // 16
	if a := polygonArea(big); a != 100 {
		t.Errorf("polygonArea(big) = %g, want 100", a)
	}
	// Point (5,5) is inside both; the tighter loop (16) wins.
	if a, ok := smallestContainingArea(m.P2(5, 5), [][]m.Point2{big, small}); !ok || a != 16 {
		t.Errorf("smallestContainingArea inside both = %g (ok=%v), want 16", a, ok)
	}
	// Point (1,1) is inside only the big loop.
	if a, ok := smallestContainingArea(m.P2(1, 1), [][]m.Point2{big, small}); !ok || a != 100 {
		t.Errorf("smallestContainingArea in big only = %g (ok=%v), want 100", a, ok)
	}
	// Point outside all loops.
	if _, ok := smallestContainingArea(m.P2(20, 20), [][]m.Point2{big, small}); ok {
		t.Errorf("smallestContainingArea outside all should be ok=false")
	}
}
