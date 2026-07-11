// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

func TestLocateEdgePicksUniqueBoxEdge(t *testing.T) {
	box, err := brep.SolidBlock(math.P3(0, 0, 0), math.P3(100, 100, 100), "box")
	if err != nil {
		t.Fatalf("box: %v", err)
	}
	// pick any real edge's midpoint as the target, then confirm round-trip.
	want := box.Edges()[3]
	ref := topo.DescribeEdge(want)
	loc := Locator{Midpoint: [3]float64{ref.Midpoint.X, ref.Midpoint.Y, ref.Midpoint.Z}}
	got, err := locateEdge(box, loc, 1e-6)
	if err != nil {
		t.Fatalf("locate: %v", err)
	}
	if got.ID() != want.ID() {
		t.Fatalf("located edge %d, want %d", got.ID(), want.ID())
	}
}
