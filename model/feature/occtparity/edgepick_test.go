// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/math"
)

// A box edge round-trips through the centroid+length locator (straight edge: centroid == mid).
func TestLocateEdgePicksUniqueBoxEdge(t *testing.T) {
	box, err := brep.SolidBlock(math.P3(0, 0, 0), math.P3(100, 100, 100), "box")
	if err != nil {
		t.Fatalf("box: %v", err)
	}
	want := box.Edges()[3]
	cen, length := edgeCentroidLength(want)
	loc := Locator{Centroid: [3]float64{cen.X, cen.Y, cen.Z}, Length: length}
	got, err := locateEdge(box, loc, 1e-6)
	if err != nil {
		t.Fatalf("locate: %v", err)
	}
	if got.ID() != want.ID() {
		t.Fatalf("located edge %d, want %d", got.ID(), want.ID())
	}
}

// Length disambiguates two edges sharing a centroid location: a locator carrying one edge's
// length must not match a co-located edge of a very different length. On a 100³ box every edge
// has length 100, so we assert the positive path plus that a wrong length is rejected.
func TestLocateEdgeLengthDisambiguates(t *testing.T) {
	box, err := brep.SolidBlock(math.P3(0, 0, 0), math.P3(100, 100, 100), "box")
	if err != nil {
		t.Fatalf("box: %v", err)
	}
	want := box.Edges()[5]
	cen, _ := edgeCentroidLength(want)
	loc := Locator{Centroid: [3]float64{cen.X, cen.Y, cen.Z}, Length: 5} // wrong length
	if _, err := locateEdge(box, loc, 1e-6); err == nil {
		t.Fatal("expected no match when the locator length disagrees with every edge")
	}
}
