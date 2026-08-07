// SPDX-License-Identifier: GPL-2.0-only

package translate

import (
	"testing"

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
