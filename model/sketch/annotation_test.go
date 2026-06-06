// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"math"
	"testing"

	gmath "oblikovati/math"
)

func TestFillRegionFindsEnclosingProfile(t *testing.T) {
	sc := NewSketches()
	s := sc.Add(XYPlane())
	s.AddRectangleByCorners(gmath.P2(0, 0), gmath.P2(10, 10))
	f := s.FillRegions().Add(gmath.P2(5, 5), "solid")
	if f.Region(s) == nil {
		t.Fatal("fill seed inside the rectangle found no region")
	}
	// A seed outside encloses nothing.
	if s.FillRegions().Add(gmath.P2(50, 50), "solid").Region(s) != nil {
		t.Fatal("fill seed outside found a region")
	}
}

func TestAnnotationsRoundTrip(t *testing.T) {
	sc := NewSketches()
	s := sc.Add(XYPlane())
	s.FillRegions().Add(gmath.P2(2, 3), "hatch")
	s.TextBoxes().Add(gmath.P2(1, 1), "PART A", 0.5, math.Pi/4, TextCenter)

	out := roundTrip(t, sc)
	if out.FillRegions().Count() != 1 || out.TextBoxes().Count() != 1 {
		t.Fatalf("after round trip: %d fills, %d texts; want 1/1", out.FillRegions().Count(), out.TextBoxes().Count())
	}
	txt := out.TextBoxes().Item(0)
	if txt.Text != "PART A" || float64(txt.Height) != 0.5 || txt.Justify != TextCenter {
		t.Fatalf("text = %+v, want 'PART A' h0.5 center", txt)
	}
	if out.FillRegions().Item(0).Style != "hatch" {
		t.Errorf("fill style = %q, want hatch", out.FillRegions().Item(0).Style)
	}
}

func TestArcSlotIsClosedProfile(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	// Centerline arc radius 5 from (5,0) to (0,5) about origin, width 2.
	ents, err := s.AddArcSlot(gmath.P2(0, 0), gmath.P2(5, 0), gmath.P2(0, 5), 2, true)
	if err != nil {
		t.Fatalf("AddArcSlot: %v", err)
	}
	if len(ents) != 4 {
		t.Fatalf("arc slot = %d entities, want 4", len(ents))
	}
	if got := s.Profiles().Count(); got != 1 {
		t.Fatalf("arc slot profiles = %d, want 1 closed region", got)
	}
}
