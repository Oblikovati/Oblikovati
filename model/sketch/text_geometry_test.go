// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"testing"

	gmath "oblikovati.org/math"
	"oblikovati.org/model/text"
)

func TestTextBoxDerivesGlyphProfiles(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	tb := s.TextBoxes().Add(gmath.P2(0, 0), "A", 1, 0, TextLeft)

	profs, err := tb.TextProfiles(text.DefaultResolver())
	if err != nil {
		t.Fatalf("TextProfiles: %v", err)
	}
	if len(profs) != 1 {
		t.Fatalf("'A' = %d profiles, want 1", len(profs))
	}
	if !profs[0].IsClosed() {
		t.Error("'A' profile should be closed")
	}
	if got := len(profs[0].InnerLoops()); got != 1 {
		t.Errorf("'A' = %d holes, want 1 (the counter)", got)
	}
}

func TestTextBoxOutlinesTranslateToAnchor(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	tb := s.TextBoxes().Add(gmath.P2(10, 20), "I", 1, 0, TextLeft)

	cs, err := tb.Outlines(text.DefaultResolver())
	if err != nil {
		t.Fatalf("Outlines: %v", err)
	}
	if len(cs) == 0 {
		t.Fatal("no contours")
	}
	// Left-baseline 'I' anchored at (10,20): all points sit at/above the anchor.
	for _, c := range cs {
		for _, p := range c {
			if float64(p.X) < 10-1e-6 || float64(p.Y) < 20-1e-6 {
				t.Fatalf("point %v below anchor (10,20)", p)
			}
		}
	}
}

func TestEmptyTextBoxHasNoGeometry(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	tb := s.TextBoxes().Add(gmath.P2(0, 0), "", 1, 0, TextLeft)
	cs, err := tb.Outlines(text.DefaultResolver())
	if err != nil {
		t.Fatalf("Outlines: %v", err)
	}
	if len(cs) != 0 {
		t.Errorf("empty text = %d contours, want 0", len(cs))
	}
}

// TestTextStyleRoundTrips proves the text entity's font/alignment fields survive a
// serialize round trip (the by-reference data, not baked geometry).
func TestTextStyleRoundTrips(t *testing.T) {
	sc := NewSketches()
	s := sc.Add(XYPlane())
	s.TextBoxes().AddStyled(gmath.P2(1, 2), "PART", 0.5, 0.1, TextCenter, TextMiddle, "Liberation Sans", 0.4)

	out := roundTrip(t, sc)
	if out.TextBoxes().Count() != 1 {
		t.Fatalf("got %d text boxes, want 1", out.TextBoxes().Count())
	}
	tb := out.TextBoxes().Item(0)
	if tb.Text != "PART" || tb.Justify != TextCenter || tb.VJustify != TextMiddle ||
		tb.Family != "Liberation Sans" || float64(tb.FontSize) != 0.4 {
		t.Errorf("round-tripped text = %+v, want PART/center/middle/Liberation Sans/0.4", tb)
	}
}
