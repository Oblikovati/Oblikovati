// SPDX-License-Identifier: GPL-2.0-only

package ipt

import "testing"

// TestExtrudeRegions pins the region decode against parts whose regions are known from how they
// were authored: each of 14_box_two's extrudes consumes its own rectangle — 4 line edges, no more,
// even though the part holds two rectangles whose associative ids both run 5..8.
func TestExtrudeRegions(t *testing.T) {
	got := ExtrudeRegions(openDoc(t, "14_box_two.ipt"))
	if len(got) != 2 {
		t.Fatalf("got %d regions, want 2", len(got))
	}
	for i, r := range got {
		if len(r) != 4 {
			t.Errorf("region %d has %d edges, want 4", i, len(r))
			continue
		}
		for _, e := range r {
			if e.Kind != EdgeLine {
				t.Errorf("region %d: edge kind %v, want a line", i, e.Kind)
			}
		}
	}
	// the two regions must be DIFFERENT rectangles: the associative ids repeat per sketch, so a
	// decode that ignored the owning sketch would return the same edges twice.
	if regionKey(got[0]) == regionKey(got[1]) {
		t.Errorf("both extrudes resolved to the same region %v — the associative id was not scoped to its sketch", regionKey(got[0]))
	}
}

// TestExtrudeRegionsRevolveDeclines: a revolve is not an extrude, so it contributes no region.
func TestExtrudeRegionsRevolveDeclines(t *testing.T) {
	if got := ExtrudeRegions(openDoc(t, "16_revolve.ipt")); len(got) != 0 {
		t.Errorf("revolve part yielded %d extrude regions, want 0", len(got))
	}
}

// regionKey is the region's bounding box — enough to tell the two rectangles apart. Summing the
// coordinates is NOT: both of 14_box_two's rectangles sum to the same value.
func regionKey(r []RegionEdge) [4]float64 {
	k := [4]float64{r[0].Line.A.X, r[0].Line.A.Y, r[0].Line.A.X, r[0].Line.A.Y}
	for _, e := range r {
		for _, p := range []Point2D{e.Line.A, e.Line.B} {
			k[0], k[1] = minf(k[0], p.X), minf(k[1], p.Y)
			k[2], k[3] = maxf(k[2], p.X), maxf(k[3], p.Y)
		}
	}
	return k
}
