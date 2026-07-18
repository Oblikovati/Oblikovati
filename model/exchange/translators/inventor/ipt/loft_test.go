// SPDX-License-Identifier: GPL-2.0-only

package ipt

import "testing"

// TestDecodeLoft checks the two-section loft: both profiles (a 2x2 and a 4x4 square) decode,
// and the section heights are (0, 4).
func TestDecodeLoft(t *testing.T) {
	d := openDoc(t, "28_loft.ipt")
	seg, _ := d.Segment("PmDCSegment")
	if s := DecodeSketches(seg); len(s) != 2 || len(s[0].Lines) != 4 || len(s[1].Lines) != 4 {
		t.Fatalf("loft sketches = %d (want 2 rectangles)", len(s))
	}
	h, ok := LoftSectionHeights(seg, 2)
	if !ok {
		t.Fatal("no loft detected")
	}
	if len(h) != 2 || h[0] != 0 || absf(h[1]-4) > 1e-6 {
		t.Errorf("section heights = %v, want (0, 4)", h)
	}
}

// TestDecodeLoft3 checks the three-section loft heights are (0, 3, 6).
func TestDecodeLoft3(t *testing.T) {
	d := openDoc(t, "31_loft3.ipt")
	seg, _ := d.Segment("PmDCSegment")
	if s := DecodeSketches(seg); len(s) != 3 {
		t.Fatalf("loft3 sketches = %d, want 3", len(s))
	}
	h, ok := LoftSectionHeights(seg, 3)
	if !ok {
		t.Fatal("no loft detected")
	}
	if len(h) != 3 || h[0] != 0 || absf(h[1]-3) > 1e-6 || absf(h[2]-6) > 1e-6 {
		t.Errorf("section heights = %v, want (0, 3, 6)", h)
	}
}

// TestHasLoftAbsent confirms non-loft parts report no loft.
func TestHasLoftAbsent(t *testing.T) {
	for _, file := range []string{"10_box.ipt", "16_revolve.ipt", "15_cylinder.ipt"} {
		d := openDoc(t, file)
		seg, _ := d.Segment("PmDCSegment")
		if _, ok := LoftSectionHeights(seg, 2); ok {
			t.Errorf("%s: decoded a loft where there is none", file)
		}
	}
}
